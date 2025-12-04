package bridge

import (
	"context"
	"sync"

	"github.com/Goden-Gun/transport-lib/pkg/envelope"
)

// Delivery wraps a transport envelope along with its ACK promise.
type Delivery struct {
	Envelope *envelope.TransportEnvelope

	ackFn func(context.Context) error

	ackOnce sync.Once
	ackErr  error
}

func newDelivery(env *envelope.TransportEnvelope, ackFn func(context.Context) error) *Delivery {
	return &Delivery{
		Envelope: env,
		ackFn:    ackFn,
	}
}

// Ack confirms the delivery back to the bridge server.
func (d *Delivery) Ack(ctx context.Context) error {
	if d == nil || d.ackFn == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	d.ackOnce.Do(func() {
		d.ackErr = d.ackFn(ctx)
	})
	return d.ackErr
}
