package ntfy

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/AnthonyHewins/gotfy"
	"github.com/charmbracelet/log"
)

type NtfyManager struct {
	NtfyServer    string
	HttpClient    *http.Client
	ntfyPublisher *gotfy.Publisher
	BaseTopic     string
	ctx           context.Context
	cancel        func()
}

func NewNtfyManager(ntfyServer string, client *http.Client) *NtfyManager {
	server, err := url.Parse(ntfyServer)
	if err != nil {
		panic(err)
	}

	if client == nil {
		client = http.DefaultClient
	}

	publisher, err := gotfy.NewPublisher(server, client)
	if err != nil {
		panic(err)
	}

	manager := &NtfyManager{
		NtfyServer:    ntfyServer,
		HttpClient:    client,
		ntfyPublisher: publisher,
	}
	manager.ctx, manager.cancel = context.WithCancel(context.Background())
	return manager
}

func (m *NtfyManager) Close() error {
	m.cancel()
	return nil
}

func (m *NtfyManager) Connect() error {
	if m.NtfyServer == "" {
		return gotfy.ErrNoServer
	}
	server, err := url.Parse(m.NtfyServer)
	if err != nil {
		return err
	}

	if m.BaseTopic == "" {
		return fmt.Errorf("BaseTopic is not set")
	}

	if m.HttpClient == nil {
		m.HttpClient = http.DefaultClient
	}

	m.ntfyPublisher, err = gotfy.NewPublisher(server, m.HttpClient)
	if err != nil {
		return err
	}

	log.Debug("Ntfy service ready to publish to", "server", m.NtfyServer)
	return nil
}

func (m *NtfyManager) Notify(message *gotfy.Message) error {
	pubResp, err := m.ntfyPublisher.SendMessage(m.ctx, message)

	if err != nil {
		return err
	}

	log.Debug("Published message", "response", pubResp)

	return nil
}
