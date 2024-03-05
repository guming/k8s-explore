package stream

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"k8s-explore/api"
	"net/http"
	"sync"
)

type MessageType string
type Message []byte

type MessageHandler interface {
	Handle(ctx context.Context, msg Message, reply chan<- Message) error
}

type Handler struct {
	api.Handler

	handlers map[MessageType]MessageHandler
	upgrader websocket.Upgrader
}

func NewHandler(logger *logrus.Entry) *Handler {
	return &Handler{
		Handler:  api.NewHandler("stream", logger),
		handlers: make(map[MessageType]MessageHandler),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (h *Handler) Connect(c *gin.Context) {
	logger := h.Logger(c).WithField("method", "connect")
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.WithError(err).Error("Couldn't upgrade to ws conn")
		c.JSON(http.StatusInternalServerError, map[string]string{"error": "upgrade failed"})
		return
	}
	newMessageDispatcher(c, conn, h.handlers, logger).dispatchLoop()

}
func (h *Handler) RegisterMessageHandler(messageType MessageType, handler MessageHandler) {
	h.handlers[messageType] = handler
}

type MessageDispatcher struct {
	ctx          context.Context
	cancel       context.CancelFunc
	conn         *websocket.Conn
	msgReadLock  sync.Mutex
	msgWriteLock sync.Mutex
	handlers     map[MessageType]MessageHandler
	logger       *logrus.Entry
}

func newMessageDispatcher(ctx context.Context, conn *websocket.Conn, handlers map[MessageType]MessageHandler,
	logger *logrus.Entry) *MessageDispatcher {
	ctx, cancel := context.WithCancel(ctx)
	return &MessageDispatcher{
		ctx:      ctx,
		cancel:   cancel,
		conn:     conn,
		handlers: handlers,
		logger:   logger,
	}
}

func (d *MessageDispatcher) dispatchLoop() {
	for {
		msg, err := d.readMessage()
		if err != nil || msg == nil {
			break
		}
		partial := struct {
			Type MessageType `json:"type"`
		}{}
		if err := json.Unmarshal(msg, partial); err != nil {
			d.logger.WithError(err).WithField("Message", msg).Warn("can not decode websocket message")
			continue
		}
		go d.dispatchMessage(msg, partial.Type)
	}
	d.cancel()
	if err := d.conn.Close(); err != nil {
		d.logger.WithError(err).Warn("Failed to close websocket connection")
	}
}

func (d *MessageDispatcher) readMessage() (Message, error) {
	d.msgReadLock.Lock()
	defer d.msgReadLock.Unlock()
	type result struct {
		data []byte
		err  error
	}
	read := make(chan result)
	//simple process
	go func() {
		msgType, data, err := d.conn.ReadMessage()
		if err != nil {
			d.logger.WithError(err).Error("Couldn't read ws message")
			read <- result{data: nil, err: err}
			return
		}
		if msgType == websocket.CloseMessage {
			d.logger.Info("ws connection has been closed")
			read <- result{data: nil, err: nil}
			return
		}
		read <- result{data: data, err: nil}
	}()
	select {
	case <-d.ctx.Done():
		return nil, nil
	case res := <-read:
		return Message(res.data), res.err
	}
}

func (d *MessageDispatcher) dispatchMessage(msg Message, msgType MessageType) {
	logger := d.logger.WithField("messageType", msgType)
	logger.Debug("Dispatching message")
	handler, found := d.handlers[msgType]
	if !found {
		logger.WithField("message", string(msg)).Warn("Unknown message type")
		return
	}
	reply := make(chan Message)
	go func() {
		defer close(reply)
		if err := handler.Handle(d.ctx, msg, reply); err != nil {
			logger.WithError(err).WithField("messageType", msgType).Warn("handle message failed")
		}
	}()
	for message := range reply {
		d.writeMessage(message)
	}
}

func (d *MessageDispatcher) writeMessage(msg Message) {
	d.msgWriteLock.Lock()
	defer d.msgWriteLock.Unlock()
	if err := d.conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
		d.logger.WithError(err).WithField("message", string(msg)).Warn("couldn't write ws message")
		d.cancel()
	}
}
