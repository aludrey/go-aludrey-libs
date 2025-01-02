package notification

type Notifier interface {
	WithPrefix(prefix string)
	Notify(msg string) error
}
