package shulker

import (
	"github.com/maddiesch/go-bus"
	"github.com/samber/do"
)

type EventBusService struct {
	*bus.Bus[Event]
}

func NewEventBusService(i *do.Injector) (*EventBusService, error) {
	return &EventBusService{
		Bus: bus.New[Event](),
	}, nil
}

func (s *EventBusService) ShutdownWithError(err error) {
	s.Bus.Publish(Event{
		Name:    EventNameShutdownError,
		Payload: err,
	})
}
