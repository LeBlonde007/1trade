package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/trade1/surveillance/internal/detect"
)

// Stream is the JetStream stream capturing both engine subjects in one order. One stream (not one per
// subject) is what lets the detectors see an order's events and its trades interleaved exactly as the
// engine emitted them.
const Stream = "TRADING_EVENTS"

// NatsPublisher publishes alerts on surveillance.alert.v1.
type NatsPublisher struct{ nc *nats.Conn }

// PublishAlert publishes one alert.
func (p NatsPublisher) PublishAlert(a detect.Alert) error {
	b, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("encode alert: %w", err)
	}
	if err := p.nc.Publish(SubjectAlerts, b); err != nil {
		return fmt.Errorf("publish alert: %w", err)
	}
	return nil
}

// Runner consumes the engine stream into a pipeline.
type Runner struct {
	nc  *nats.Conn
	sub *nats.Subscription
}

// Start connects, ensures the stream exists, and replays it FROM THE BEGINNING through the pipeline
// with an ordered consumer, then keeps following it live. Replaying rather than resuming is deliberate:
// detector state (windows, net positions) lives in memory and positions accumulate from the first
// trade, so it is rebuilt on every start. Deterministic alert ids + idempotent storage make the replay
// silent. The pipeline is built by mk with the connection's publisher.
func Start(url string, mk func(Publisher) *Pipeline) (*Runner, error) {
	nc, err := nats.Connect(url, nats.Name("surveillance"), nats.Timeout(5*time.Second))
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("jetstream: %w", err)
	}
	if _, err := js.StreamInfo(Stream); err != nil {
		if _, err := js.AddStream(&nats.StreamConfig{
			Name: Stream, Subjects: []string{SubjectOrders, SubjectTrades}, Storage: nats.FileStorage,
		}); err != nil {
			nc.Close()
			return nil, fmt.Errorf("add stream: %w", err)
		}
	}
	p := mk(NatsPublisher{nc: nc})
	msgs := make(chan *nats.Msg, 1024)
	sub, err := js.ChanSubscribe(">", msgs, nats.BindStream(Stream), nats.OrderedConsumer(), nats.DeliverAll())
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("subscribe: %w", err)
	}
	go func() {
		for m := range msgs {
			handle(p, m)
		}
	}()
	slog.Info("surveillance consuming", "stream", Stream)
	return &Runner{nc: nc, sub: sub}, nil
}

// handle runs one message, retrying a failing store a few times. An undecodable message is logged and
// skipped. Ordered consumers need no ack.
func handle(p *Pipeline, m *nats.Msg) {
	for attempt := 1; ; attempt++ {
		_, err := p.Handle(context.Background(), m.Subject, m.Data)
		if err == nil {
			return
		}
		var syntax *json.SyntaxError
		if errors.As(err, &syntax) || attempt >= 5 {
			slog.Error("surveillance: message not processed", "subject", m.Subject, "err", err)
			return
		}
		time.Sleep(time.Duration(attempt) * 200 * time.Millisecond)
	}
}

// Close stops consuming and closes the connection.
func (r *Runner) Close() {
	_ = r.sub.Unsubscribe()
	_ = r.nc.Drain()
}
