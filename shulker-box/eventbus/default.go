package eventbus

// Default is the shared default event bus
var Default = New()

func Subscribe(name ...string) Subscription {
	return Default.Subscribe(name...)
}

func Dispatch(name string, val interface{}) {
	Default.Dispatch(name, val)
}

func Listen(name string, fn func(e interface{}), closed ...func()) Subscription {
	return Default.Listen(name, fn, closed...)
}
