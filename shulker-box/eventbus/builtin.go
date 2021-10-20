package eventbus

import (
	"os"
	"os/signal"
)

const SignalEvent = `eventbus.SignalEvent`

func InstallSignalEvent(b *EventBus, sig ...os.Signal) {
	go func() {
		interuptChan := make(chan os.Signal, 1)

		signal.Notify(interuptChan, sig...)

		sig := <-interuptChan

		b.Dispatch(SignalEvent, sig)
	}()
}
