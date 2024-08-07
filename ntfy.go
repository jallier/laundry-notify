package laundryNotify

type LaundryNotifyService interface {
	Notify(topic string, title string, message string) error
}
