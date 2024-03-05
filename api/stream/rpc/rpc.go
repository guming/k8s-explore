package rpc

import (
	"context"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"k8s-explore/api/stream"
	"k8s-explore/logging"
	"sync"
)

const (
	MessageTypeCall stream.MessageType = "call"
)

type CallID string

type CallMethod string

type Call struct {
	ID     CallID          `json:"id"`
	Method CallMethod      `json:"method"`
	Params json.RawMessage `json:"params"`
}

const (
	callMethodCancel CallMethod = ".cancel"
)

type CallHandler interface {
	Handle(ctx context.Context, call Call, reply chan<- stream.Message) error
}

type CallDispatcher struct {
	handlers    map[CallMethod]CallHandler
	activeCalls map[CallID]context.CancelFunc
	activeLock  sync.Mutex
	logger      *logrus.Entry
}

func NewCallDispatcher() *CallDispatcher {
	return &CallDispatcher{
		handlers:    make(map[CallMethod]CallHandler),
		activeCalls: make(map[CallID]context.CancelFunc),
		activeLock:  sync.Mutex{},
		logger:      logrus.WithField("module", "stream/rpc/callDispatcher"),
	}
}

func (d *CallDispatcher) RegisterCallHandler(method CallMethod, handler CallHandler) {
	d.handlers[method] = handler
}

func (d *CallDispatcher) Handle(ctx context.Context, msg stream.Message, reply chan<- stream.Message) error {
	logger := logging.WithRequestID(ctx, d.logger)
	call := Call{}
	if err := json.Unmarshal(msg, &call); err != nil {
		logger.WithError(err).WithField("message", msg).Warn("Couldn't decode stream message into RPC call")
		return err
	}
	logger = logger.
		WithField("callId", call.ID).
		WithField("callMethod", call.Method)
	logger.Debug("Dispatching RPC call")
	if call.Method == callMethodCancel {
		d.activeLock.Lock()
		defer d.activeLock.Unlock()
		if cancel, found := d.activeCalls[call.ID]; found {
			cancel()
			delete(d.activeCalls, call.ID)
		} else {
			logger.Warn("active rpc call not found - nothing to cancel.")
		}
		reply <- okReply(call)
		return nil
	}

	handler, found := d.handlers[call.Method]
	if !found {
		logger.Debug("RPC call handler not found")
		reply <- koReply(call, "Unknown method")
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)

	d.activeLock.Lock()
	d.activeCalls[call.ID] = cancel
	d.activeLock.Unlock()

	err := handler.Handle(ctx, call, reply)

	d.activeLock.Lock()
	delete(d.activeCalls, call.ID)
	d.activeLock.Unlock()

	return err
}

func okReply(call Call) stream.Message {
	msg, err := json.Marshal(map[string]interface{}{"id": call.ID, "result": "ok"})
	if err != nil {
		panic(err)
	}
	return stream.Message(msg)
}

func koReply(call Call, errMsg string) stream.Message {
	msg, err := json.Marshal(map[string]interface{}{"id": call.ID, "error": errMsg})
	if err != nil {
		panic(err)
	}
	return stream.Message(msg)
}
