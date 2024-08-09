package main

import (
	"jallier/laundry-notify/internal/http"
	"jallier/laundry-notify/internal/mqtt"
	"jallier/laundry-notify/internal/ntfy"
	"jallier/laundry-notify/internal/sqlite"
	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"
	"golang.org/x/net/context"
)

func main() {
	log.Info("starting....")

	// Setup signal handlers
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
	}()

	m := NewMain()

	// parse env vars and load config
	err := godotenv.Load("data/.env")
	if err != nil {
		log.Debug("Error loading .env file", "error", err)
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
	log.Info("application set up and started... Listening for events")

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
	Ntfy   *ntfy.NtfyManager
	Http   *http.HttpServer
	Config *Config
}

// Returns a new instance of Main
func NewMain() *Main {
	return &Main{
		DB:     sqlite.NewDB(""),
		MQTT:   mqtt.NewMQTTManager(),
		Ntfy:   ntfy.NewNtfyManager("", nil),
		Http:   http.NewHttpServer(),
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

	if m.Config.Ntfy.NtfyServer == "" {
		m.Config.Ntfy.NtfyServer = "https://ntfy.sh"
	}
	m.Ntfy.NtfyServer = m.Config.Ntfy.NtfyServer
	m.Ntfy.BaseTopic = m.Config.Ntfy.BaseTopic
	err = m.Ntfy.Connect()
	if err != nil {
		log.Error("failed to connect to ntfy server", "error", err)
		return err
	}

	m.Http.Config.Env = m.Config.Http.Env
	m.Http.Open()

	// Set up the services using the root dependencies
	// userService := sqlite.NewUserService(m.DB)
	eventService := sqlite.NewEventService(m.DB)
	userEventService := sqlite.NewUserEventService(m.DB)

	ntfyService := ntfy.NewLaundryNotifyService(m.Ntfy)

	laundrySubscriberService := mqtt.NewLaundrySubscriberService(
		m.MQTT,
		eventService,
		userEventService,
		ntfyService,
	)
	laundrySubscriberService.Subscribe(m.Config.MQTT.topic)

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
	Ntfy struct {
		NtfyServer string
		BaseTopic  string
	}
	Http struct {
		Env string
	}
	Env string
}

// DefaultConfig returns a new instance of Config with default values
func DefaultConfig() *Config {
	var config Config
	config.DB.DSN = DefaultDSN

	return &config
}

func SetConfigFromEnv(config *Config) {
	config.Env = os.Getenv("ENV")
	config.DB.DSN = os.Getenv("DB_DSN")
	if config.DB.DSN == "" {
		log.Fatal("DB_DSN is required")
	}
	config.MQTT.URL = os.Getenv("MQTT_URL")
	if config.MQTT.URL == "" {
		log.Fatal("MQTT_URL is required")
	}
	config.MQTT.ClientId = os.Getenv("MQTT_CLIENT_ID")
	if config.MQTT.ClientId == "" {
		log.Fatal("MQTT_CLIENT_ID is required")
	}
	config.MQTT.Username = os.Getenv("MQTT_USERNAME")
	config.MQTT.Password = os.Getenv("MQTT_PASSWORD")
	config.MQTT.topic = os.Getenv("MQTT_TOPIC")
	if config.MQTT.topic == "" {
		log.Fatal("MQTT_TOPIC is required")
	}
	config.Ntfy.NtfyServer = os.Getenv("NTFY_SERVER")
	config.Ntfy.BaseTopic = os.Getenv("NTFY_BASE_TOPIC")
	if config.Ntfy.BaseTopic == "" {
		log.Fatal("NTFY_BASE_TOPIC is required")
	}
	config.Http.Env = config.Env
}
