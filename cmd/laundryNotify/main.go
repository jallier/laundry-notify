package main

import (
	"jallier/laundry-notify/internal/sqlite"
	"log"
	"os"
	"os/signal"

	"golang.org/x/net/context"
)

func main() {
	// Setup signal handlers
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() { <-c; cancel() }()

	m := NewMain()

	if err := m.Run(ctx); err != nil {
		log.Printf("error: %v", err)
		m.Close()
		os.Exit(1)
	}

	// Wait for ctrl c
	<-ctx.Done()

	if err := m.Close(); err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
}

type Main struct {
	DB *sqlite.DB
}

func NewMain() *Main {
	return &Main{
		DB: sqlite.NewDB(""),
	}
}

func (m *Main) Close() error {
	if m.DB != nil {
		if err := m.DB.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (m *Main) Run(ctx context.Context) (err error) {
	return nil
}
