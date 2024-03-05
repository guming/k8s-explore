package logging

import (
	"context"
	"github.com/sirupsen/logrus"
)

type loggingContextKey int

const KeyRequestID loggingContextKey = iota

func WithRequestID(cxt context.Context, logger *logrus.Entry) *logrus.Entry {
	rid, ok := cxt.Value(KeyRequestID).(string)
	if !ok {
		rid = "none"
	}
	return logger.WithField("request_id", rid)
}
