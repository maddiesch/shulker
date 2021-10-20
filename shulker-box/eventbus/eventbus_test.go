package eventbus_test

import (
	"shulker-box/eventbus"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventBus(t *testing.T) {
	bus := eventbus.New()
	defer bus.Close()

	t.Run(`Subscribe`, func(t *testing.T) {
		s := bus.Subscribe(`test_1`)

		assert.NoError(t, s.Close())
	})

	t.Run(`Listen`, func(t *testing.T) {
		d := make(chan struct{})

		s := bus.Listen(`test_1`, func(_ interface{}) {
			close(d)
		})
		defer s.Close()

		bus.Dispatch(`test_1`, nil)

		<-d
	})
}
