package ntfy

import (
	laundryNotify "jallier/laundry-notify"

	"github.com/AnthonyHewins/gotfy"
)

var _ laundryNotify.LaundryNotifyService = (*LaundryNotifyService)(nil)

type LaundryNotifyService struct {
	ntfyManager *NtfyManager
}

func NewLaundryNotifyService(ntfyManager *NtfyManager) *LaundryNotifyService {
	return &LaundryNotifyService{ntfyManager: ntfyManager}
}

func (s *LaundryNotifyService) Notify(topic string, title string, message string) error {
	fullTopic := s.ntfyManager.BaseTopic + "-" + topic
	messageStruct := &gotfy.Message{
		Topic:   fullTopic,
		Title:   title,
		Message: message,
	}

	return s.ntfyManager.Notify(messageStruct)
}
