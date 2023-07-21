package shulker

type EventName string

const (
	EventNameShutdown      EventName = "shulker.shutdown"
	EventNameShutdownError EventName = "shulker.shutdown-error"
)

type Event struct {
	Name    EventName
	Payload any
}
