package mocks

import (
	"context"
)

// MockEventProducer is a mock implementation of EventProducer
type MockEventProducer struct {
	ProduceFunc  func(ctx context.Context, key, value []byte) error
	ProduceCalls []ProduceCall
	Closed       bool
}

type ProduceCall struct {
	Key   []byte
	Value []byte
}

func (m *MockEventProducer) Produce(ctx context.Context, key, value []byte) error {
	m.ProduceCalls = append(m.ProduceCalls, ProduceCall{Key: key, Value: value})
	if m.ProduceFunc != nil {
		return m.ProduceFunc(ctx, key, value)
	}
	return nil
}

func (m *MockEventProducer) Close() error {
	m.Closed = true
	return nil
}
