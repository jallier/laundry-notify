package main

import (
	"jallier/laundry-notify/internal/mqtt"
	"jallier/laundry-notify/internal/sqlite"
	"os"
	"os/signal"

	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"
	"golang.org/x/net/context"
)

func main() {
	log.Info("starting...")

	// Setup signal handlers
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() { <-c; cancel() }()

	m := NewMain()

	// parse env vars and load config
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file. Either one not provided or running in prod mode")
	}
	env := os.Getenv("ENV")
	if env == "dev" || env == "development" {
		log.SetLevel(log.DebugLevel)
	}
	SetConfigFromEnv(m.Config)

	// Execute the application
	if err := m.Run(ctx); err != nil {
		log.Printf("error: %v", err)
		m.Close()
		os.Exit(1)
	}

	// Wait for ctrl c
	<-ctx.Done()
	log.Info("shutting down...")

	if err := m.Close(); err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
}

type Main struct {
	DB     *sqlite.DB
	MQTT   *mqtt.MQTTManager
	Config *Config
}

// Returns a new instance of Main
func NewMain() *Main {
	return &Main{
		DB:     sqlite.NewDB(""),
		MQTT:   mqtt.NewMQTTManager(),
		Config: DefaultConfig(),
	}
}

// Close gracefully shuts down the application
func (m *Main) Close() error {
	if m.DB != nil {
		if err := m.DB.Close(); err != nil {
			return err
		}
	}

	if m.MQTT != nil {
		m.MQTT.Disconnect()
	}

	return nil
}

// Run starts the application. The config must be loaded before calling this method
func (m *Main) Run(ctx context.Context) (err error) {
	log.Debug("config: ", "config", m.Config)

	// Set up the main root dependencies
	m.DB.DSN = m.Config.DB.DSN
	if err := m.DB.Open(); err != nil {
		log.Error("failed to open db", "error", err)
		return err
	}

	mqttOpts := mqtt.NewMqttOpts()
	mqttOpts.AddBroker(m.Config.MQTT.URL)
	mqttOpts.SetClientID(m.Config.MQTT.ClientId)
	mqttOpts.SetUsername(m.Config.MQTT.Username)
	mqttOpts.SetPassword(m.Config.MQTT.Password)

	m.MQTT.MqttOpts = mqttOpts
	_, err = m.MQTT.Connect()
	if err != nil {
		log.Error("failed to connect to mqtt broker", "error", err)
		return err
	}

	// Set up the services using the root dependencies
	userService := sqlite.NewUserService(m.DB)
	eventService := sqlite.NewEventService(m.DB)
	userEventService := sqlite.NewUserEventService(m.DB)

	laundrySubscriberService := mqtt.NewLaundrySubscriberService(
		m.MQTT,
		eventService,
	)
	laundrySubscriberService.Subscribe(m.Config.MQTT.topic)

	// This is just testing for now
	user, err := userService.FindUserById(ctx, 1)
	if err != nil {
		log.Error("failed to find user", "error", err)
	}
	log.Debug("user", "user", user)

	// create := &laundryNotify.Event{
	// 	Type:       "washer",
	// 	StartedAt:  time.Now(),
	// 	FinishedAt: time.Now(),
	// }
	// eventService.CreateEvent(ctx, create)

	event, err := eventService.FindEventById(ctx, 1)
	if err != nil {
		log.Error("failed to find event", "error", err)
	}
	log.Debug("event", "event", event)

	// create2 := &laundryNotify.UserEvent{
	// 	UserId:    user.Id,
	// 	EventId:   event.Id,
	// 	CreatedAt: time.Now(),
	// }
	// err = userEventService.CreateUserEvent(ctx, create2)
	if err != nil {
		log.Error("failed to create user event", "error", err)
	}
	userEvent, err := userEventService.FindUserEventById(ctx, 1)
	if err != nil {
		log.Error("failed to find user event", "error", err)
	}
	log.Debug("user event", "user event", userEvent)

	return nil
}

const DefaultDSN = "data.db"

// Config represents the application configuration
type Config struct {
	MQTT struct {
		URL      string
		ClientId string
		Username string
		Password string
		topic    string
	}
	DB struct {
		DSN string
	}
}

// DefaultConfig returns a new instance of Config with default values
func DefaultConfig() *Config {
	var config Config
	config.DB.DSN = DefaultDSN

	return &config
}

func SetConfigFromEnv(config *Config) {
	config.DB.DSN = os.Getenv("DB_DSN")
	config.MQTT.URL = os.Getenv("MQTT_URL")
	config.MQTT.ClientId = os.Getenv("MQTT_CLIENT_ID")
	config.MQTT.Username = os.Getenv("MQTT_USERNAME")
	config.MQTT.Password = os.Getenv("MQTT_PASSWORD")
	config.MQTT.topic = os.Getenv("MQTT_TOPIC")
}
