package eventbus

import (
	"sync"
)

type Subscription interface {
	Close() error
	Receive() <-chan Event
	Done() <-chan struct{}
	IsClosed() bool
}

type subscription struct {
	mu     sync.Mutex
	bus    *EventBus
	id     uint64
	rec    chan Event
	closer chan struct{}
}

func (s *subscription) IsClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.closer == nil
}

func (s *subscription) Close() error {
	if b := s.bus; b != nil {
		b.delete(s.id)
	}

	s.finalize()

	return nil
}

func (s *subscription) finalize() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closer != nil {
		close(s.closer)
	}
	if s.rec != nil {
		close(s.rec)
	}
	s.rec = nil
	s.closer = nil
}

func (s *subscription) Receive() <-chan Event {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.rec
}

func (s *subscription) Done() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closer == nil {
		panic("attempting to wait for closing of an already closed subscription")
	}

	return s.closer
}

func (s *subscription) dispatch(e Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.rec == nil {
		return
	}

	s.rec <- e
}

var _ Subscription = new(subscription)
