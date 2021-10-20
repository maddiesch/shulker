package eventbus

import (
	"sync"
	"sync/atomic"
)

type Event interface {
	Name() string
	Value() interface{}
}

type EventBus struct {
	mu   sync.RWMutex
	id   uint64
	subs map[string][]*subscription
}

// New returns a new EventBus
func New() *EventBus {
	return &EventBus{
		subs: make(map[string][]*subscription),
	}
}

// Close removes all subscriptions. After calling close the EventBus can still
// be used to re-subscribe and dispatch events.
func (b *EventBus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, subs := range b.subs {
		for _, s := range subs {
			s.finalize()
		}
	}

	b.subs = make(map[string][]*subscription)

	return nil
}

// Subscribe creates a new subscription listening for the given event names.
func (e *EventBus) Subscribe(name ...string) Subscription {
	e.mu.Lock()
	defer e.mu.Unlock()

	id := atomic.AddUint64(&e.id, 1)

	sub := &subscription{
		bus:    e,
		id:     id,
		rec:    make(chan Event, 1),
		closer: make(chan struct{}),
	}

	for _, n := range name {
		col := e.subs[n]
		e.subs[n] = append(col, sub)
	}

	return sub
}

// Dispatch sends a new event with the passed name and value. Value can be nil
// if listeners do not expect any values.
func (b *EventBus) Dispatch(name string, val interface{}) {
	b.DispatchEvent(&event{name, val})
}

func (b *EventBus) DispatchEvent(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if col, ok := b.subs[event.Name()]; ok {
		for _, sub := range col {
			go sub.dispatch(event)
		}
	}
}

// Listen creates a subscription and begins handling events on a goroutine.
func (b *EventBus) Listen(name string, fn func(interface{}), closed ...func()) Subscription {
	sub := b.Subscribe(name)

	go func() {
		for {
			if sub.IsClosed() {
				return
			}

			select {
			case <-sub.Done():
				for _, c := range closed {
					c()
				}
				return
			case e := <-sub.Receive():
				if e != nil {
					fn(e.Value())
				}
			}
		}
	}()

	return sub
}

func (e *EventBus) delete(i uint64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for key := range e.subs {
		col := e.subs[key]
		rep := make([]*subscription, 0, len(col))

		for _, sub := range col {
			if sub.id != i {
				rep = append(rep, sub)
			}
		}

		e.subs[key] = rep
	}
}

type event struct {
	name  string
	value interface{}
}

func (d *event) Name() string       { return d.name }
func (d *event) Value() interface{} { return d.value }

var _ Event = new(event)
