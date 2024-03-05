package notify_test

import (
	"github.com/stretchr/testify/assert"
	"k8s-explore/frp/notify"
	"testing"
)

func TestSignal_HasNotified(t *testing.T) {

	signal := notify.NewCancellationChannel()

	assert.False(t, signal.IsCancelled())

	signal.Cancel()

	assert.True(t, signal.IsCancelled())
}

func TestSignal_CancelledTwice(t *testing.T) {

	signal := notify.NewCancellationChannel()

	signal.Cancel()
	signal.Cancel()
	assert.True(t, signal.IsCancelled())
}
