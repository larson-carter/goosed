package bus

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"

	"github.com/nats-io/nats.go"
)

// Bus wraps a NATS JetStream connection for publishing and consuming events.
type Bus struct {
	conn *nats.Conn
	js   nats.JetStreamContext
}

// New creates a Bus connected to the provided NATS endpoint.
func New(url string, opts ...nats.Option) (*Bus, error) {
	nc, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, err
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, err
	}

	return &Bus{conn: nc, js: js}, nil
}

// Close shuts down the underlying NATS connection.
func (b *Bus) Close() {
	if b == nil {
		return
	}
	if err := b.conn.Drain(); err != nil {
		b.conn.Close()
	}
}

// Publish encodes v as JSON and publishes it to the given subject.
func (b *Bus) Publish(ctx context.Context, subj string, v any) error {
	if b == nil {
		return errors.New("nil bus")
	}

	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	_, err = b.js.Publish(subj, data, nats.Context(ctx))
	return err
}

type subscription struct {
	sub    *nats.Subscription
	mu     sync.Mutex
	closed bool
}

func (s *subscription) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	return s.sub.Drain()
}

// Subscribe creates a durable consumer on the given subject and invokes fn for each message.
func (b *Bus) Subscribe(ctx context.Context, subj, durable string, fn func(ctx context.Context, data []byte) error) (io.Closer, error) {
	if b == nil {
		return nil, errors.New("nil bus")
	}
	if fn == nil {
		return nil, errors.New("nil handler")
	}

	handler := func(msg *nats.Msg) {
		handlerCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		if err := fn(handlerCtx, msg.Data); err != nil {
			_ = msg.Nak()
			return
		}
		_ = msg.Ack()
	}

	sub, err := b.js.Subscribe(subj, handler, nats.Durable(durable), nats.ManualAck(), nats.AckExplicit())
	if err != nil {
		return nil, err
	}

	s := &subscription{sub: sub}

	go func() {
		<-ctx.Done()
		_ = s.Close()
	}()

	return s, nil
}
