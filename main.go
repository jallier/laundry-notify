package laundryNotify

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/foolin/goview/supports/ginview"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// What needs to happen:
// A user registers their interest with their name and is given the ntfy topic link
// the app subscribes to laundry events on mqtt
// Home assistant will publish messages to this topic
// If a user has registered interest, send the next 'finished' message to the ntfy topic + the users name
// Delete the user's "interest"

// The base URL for ntfy.sh
const NTFY_URL = "https://ntfy.sh/LaundryTest"

var db *sql.DB
var isDev bool
var topic string

func main() {
	// Setup stuff up here
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file. Either one not provided or running in prod mode")
	}
	env := getEnv("ENV")
	isDev = false
	if env == "dev" || env == "development" {
		log.SetLevel(log.DebugLevel)
		isDev = true
	}
	topic = getEnv("MQTT_TOPIC")
	mqttUrl := getEnv("MQTT_URL")
	mqttClientId := getEnv("MQTT_CLIENT_ID")
	mqttUsername := getEnv("MQTT_USERNAME")
	mqttPassword := getEnv("MQTT_PASSWORD")
	if topic == "" {
		log.Fatal("MQTT_TOPIC env var not set; Please set this in the .env file and restart the application")
	}
	log.Info("Starting application")
	log.Info("Log set to " + log.GetLevel().String())

	go setupRouter()

	db, err = sql.Open("sqlite3", "data.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	sqlStmt := `
	create table if not exists users (
		id integer not null primary key,
		name text unique,
		created_at datetime not null
	);

	create table if not exists events (
		id integer not null primary key,
		type text not null,
		started_at datetime not null,
		finished_at datetime
	);

	create table if not exists user_events (
		id integer not null primary key,
		user_id integer not null,
		event_id integer,
		created_at datetime not null,
		UNIQUE (user_id, event_id)
	);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}
	log.Debug("Database setup")

	opts := MQTT.NewClientOptions()
	opts.AddBroker(mqttUrl)
	opts.SetClientID(mqttClientId)
	opts.SetUsername(mqttUsername)
	opts.SetPassword(mqttPassword)

	log.Info("starting subscriber")
	events := make(chan [2]string)

	opts.SetDefaultPublishHandler(func(client MQTT.Client, msg MQTT.Message) {
		events <- [2]string{msg.Topic(), string(msg.Payload())}
	})

	client := MQTT.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal("error establishing mqtt connection", "error", token.Error())
	}
	log.Info("Connected")

	if token := client.Subscribe(topic, byte(0), nil); token.Wait() && token.Error() != nil {
		log.Error("error establishing connection to topic", "error", token.Error(), "topic", topic)
		os.Exit(1)
	}
	log.Info("Subscribed to topic", "topic", topic)

	// Write the events to the db as they come in from the channel
	for incomingEvent := range events {
		topic, message := incomingEvent[0], incomingEvent[1]
		log.Debug("RECEIVED TOPIC: %s MESSAGE: %s\n", "topic", topic, "message", message)

		topicSlice := strings.Split(topic, "/")
		leafTopic := topicSlice[len(topicSlice)-1]

		messageSlice := strings.Split(message, "=")
		messageKey := messageSlice[0]
		messageValue := messageSlice[1]

		tx, err := db.Begin()
		if err != nil {
			log.Error("Error starting transaction", "error", err)
			continue
		}

		if messageKey == "started_at" {
			stmt, err := tx.Prepare("insert into events (type, started_at) values (?, ?)")
			if err != nil {
				log.Error("Error preparing statement", "error", err)
				continue
			}
			_, err = stmt.Exec(leafTopic, messageValue)
			if err != nil {
				log.Error("Error inserting event", "error", err)
				continue
			}
			// stmt.Close()
			log.Info("New event written to db", "topic", leafTopic, "message", message)
		}
		if messageKey == "finished_at" {
			// Get the most recent unfinished event
			stmtQuery, err := tx.Prepare("select id from events where type = ? and finished_at is null order by started_at desc limit 1")
			if err != nil {
				log.Fatal("Error preparing statement", "error", err)
			}
			var eventId int
			err = stmtQuery.QueryRow(leafTopic).Scan(&eventId)
			if err != nil {
				log.Error("Error querying for unfinished event", "error", err)
				continue
			}

			// Update the most recent unfinished event
			stmt, err := tx.Prepare("update events set finished_at = ? where id = ?")
			if err != nil {
				log.Error("Error preparing statement", "error", err)
				continue
			}
			_, err = stmt.Exec(messageValue, eventId)
			if err != nil {
				log.Error("Error updating event", "error", err)
				continue
			}
			log.Info("Event updated in db", "id", eventId, "topic", leafTopic, "message", message)

			// Check the user events for any users registered for this event
			rows, err := tx.Query("select users.name from users join user_events on users.id = user_events.user_id where user_events.event_id = ?", eventId)
			if err != nil {
				log.Error("Error querying for users registered for event", "error", err)
				continue
			}
			for rows.Next() {
				var user string
				err := rows.Scan(&user)
				if err != nil {
					log.Error("Error scanning user record", "error", err)
					continue
				}
				log.Debug("Sending notification to user", "user", user)
				// Convert the username to something that can be used in a URL
				username := strings.ReplaceAll(user, " ", "-")
				ntfyFullUrl := fmt.Sprintf("%s-%s", NTFY_URL, username)
				title := toTitleCase(leafTopic)
				message := fmt.Sprintf("%s has finished", toTitleCase(leafTopic))
				req, _ := http.NewRequest(
					"POST",
					ntfyFullUrl,
					strings.NewReader(message),
				)
				req.Header.Set("Title", title)
				res, err := http.DefaultClient.Do(req)
				if err != nil {
					log.Error("Error sending notification", "error", err)
					continue
				}
				if res.StatusCode != 200 {
					log.Error("Error sending notification", "status", res.StatusCode)
					continue
				}
				log.Info("Notification sent", "user", user, "event", leafTopic, "ntfy_url", ntfyFullUrl)
				res.Body.Close()
			}
		}

		err = tx.Commit()
		if err != nil {
			log.Error("Error writing event to db", "error", err)
		} else {
			log.Debug("Transaction committed")
		}
	}

}

// Get an env var
func getEnv(envVar string) string {
	return os.Getenv(envVar)
}

func setupRouter() {
	router := gin.Default()
	if isDev {
		router.ForwardedByClientIP = true
		router.SetTrustedProxies([]string{"127.0.0.1"})
	}
	router.HTMLRender = ginview.Default()
	router.Static("/static", "./static")

	router.GET("/ping", getPing)
	router.GET("/", handleIndex)
	router.POST("/search", handleSearch)
	register := router.Group("/register")
	{
		register.POST("/", handleRegister)
		register.GET("/", handleRegister)
	}

	router.Run()
}

// Simple ping function
func getPing(c *gin.Context) {
	c.String(http.StatusOK, "PONG")
}

func getRecentUsers() ([]string, error) {
	rows, err := db.Query("select name from users order by created_at desc limit 1")
	if err != nil {
		log.Error("Error querying users table")
		return nil, err
	}
	defer rows.Close()

	var users []string
	for rows.Next() {
		var user string
		err := rows.Scan(&user)
		if err != nil {
			log.Error("Error scanning user record", "error", err)
			continue
		}
		users = append(users, user)
	}

	return users, nil
}

func getMostRecentEvent() (*Event, error) {
	row := db.QueryRow("SELECT id, type, started_at, finished_at FROM events ORDER BY MAX(started_at, IFNULL(finished_at, started_at)) DESC")
	var event Event
	err := row.Scan(&event.Id, &event.Type, &event.StartedAt, &event.FinishedAt)
	if err != nil {
		log.Error("Error scanning event record", "error", err)
		return &Event{}, err
	}

	return &event, nil
}

func handleIndex(c *gin.Context) {
	users, err := getRecentUsers()
	if err != nil {
		log.Error("Error getting recent users", "error", err)
		users = []string{}
	}
	mostRecentEvent, err := getMostRecentEvent()
	if err != nil {
		log.Error("Error getting most recent event", "error", err)
		mostRecentEvent = &Event{}
	}
	log.Debug("Most recent event", "event", mostRecentEvent)
	c.HTML(http.StatusOK, "index", gin.H{
		"title":           "Laundry Notifications",
		"users":           users[:1],
		"mostRecentEvent": mostRecentEvent,
	})
}

type SearchRequest struct {
	Name string `form:"name"`
}

func handleSearch(c *gin.Context) {
	var req SearchRequest
	c.Bind(&req)
	log.Debug("Received search request", "name", req.Name)
	users := []string{}

	// If no name is provided, return the most recent user
	if req.Name == "" {
		users, err := getRecentUsers()
		if err != nil {
			log.Error("Error getting recent users", "error", err)
			users = []string{}
		}
		c.HTML(http.StatusOK, "partials/search.html", gin.H{
			"users": users,
		})
		return
	}

	row := db.QueryRow("select name from users where name = ?", req.Name)
	var name string
	err := row.Scan(&name)
	if err != nil {
		log.Error("Error scanning user record", "error", err)
	} else {
		log.Debug("User found", "name", name)
		users = []string{name}
	}

	c.HTML(http.StatusOK, "partials/search.html", gin.H{
		"users": users,
	})
}

func handleRegister(c *gin.Context) {
	var req SearchRequest
	c.Bind(&req)
	log.Debug("Received add user request", "name", req.Name)

	// Check if the user already exists
	row := db.QueryRow("select name from users where name = ?", req.Name)
	var name string
	err := row.Scan(&name)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Fatal("Error scanning user record", "error", err)
		}

		log.Debug("User does not exist")
	}

	if name != "" {
		log.Debug("User already exists", "name", name)
	} else {
		log.Debug("User does not exist, adding", "name", name)
		_, err = db.Exec("insert into users (name, created_at) values (?, datetime())", req.Name)
		if err != nil {
			// TODO: add a flash message to the page for errors
			log.Error("Error adding user", "error", err)
			c.HTML(http.StatusOK, "register", gin.H{
				"error": err,
			})
		}
		log.Debug("User added", "name", name)
	}
	log.Info("Registering user interest")

	// First get the most recent event
	event, err := getMostRecentEvent()
	if err != nil {
		if err != sql.ErrNoRows {
			log.Error("Error getting most recent event", "error", err)
			c.HTML(http.StatusOK, "registered", gin.H{
				"error": err,
			})
			return
		}
	}
	// Now register the user for this event
	// If the event is finished, register with a blank finished_at because we want to register for the next one
	if !event.FinishedAt.Time.IsZero() {
		log.Debug("Event is finished, registering for next one")
		// First check if they have already registered for the next event
		row := db.QueryRow("select id from user_events where user_id = (select id from users where name = ?) and event_id is null", name)
		var id int
		err := row.Scan(&id)
		if err != nil {
			if err != sql.ErrNoRows {
				log.Error("Error scanning user event record", "error", err)
				c.HTML(http.StatusOK, "registered", gin.H{
					"error": err,
				})
				return
			} else {
				log.Debug("User has not registered for next event")
			}
		} else {
			log.Debug("User has already registered for next event", "id", id)
			c.HTML(http.StatusOK, "registered", gin.H{
				"name":  name,
				"event": event,
			})
			return
		}

		_, err = db.Exec("insert into user_events (user_id, created_at) values ((select id from users where name = ?), datetime())", name)
		if err != nil {
			log.Error("Error registering user for event", "error", err)
			c.HTML(http.StatusOK, "registered", gin.H{
				"error": err,
			})
			return
		}
	} else {
		log.Debug("Event is not finished, registering for this one")
		// TODO: replace datetimes like this with ISO 8601 format
		_, err = db.Exec("insert into user_events (user_id, event_id, created_at) values ((select id from users where name = ?), ?, datetime())", name, event.Id)
		if err != nil {
			log.Error("Error registering user for event", "error", err)
			c.HTML(http.StatusOK, "registered", gin.H{
				"error": err,
			})
			return
		}
	}
	log.Info("User registered for event", "name", name, "event", event)

	c.HTML(http.StatusOK, "registered", gin.H{
		"name":  name,
		"event": event,
	})
}

func toTitleCase(s string) string {
	return cases.Title(language.English).String(s)
}

type EventFilter struct {
	Id         *int
	Type       *string
	StartedAt  time.Time
	FinishedAt time.Time
	Limit      int
	Offset     int
	OrderBy    []string
}

type EventService interface {
	FindEventById(ctx context.Context, userId int) (*Event, error)
	FindMostRecentEvent(ctx context.Context, eventType string) (*Event, error)
	CreateEvent(ctx context.Context, event *Event) error
	UpdateEvent(ctx context.Context, id int, update EventUpdate) (*Event, error)
}

type UserEventFilter struct {
	Id *int
}

type UserEventService interface {
	FindUserEventById(ctx context.Context, id int) (*UserEvent, error)
	FindUserNamesByEventId(ctx context.Context, eventId int) ([]string, error)
	FindByUserName(ctx context.Context, name string, eventType string) ([]*UserEvent, int, error)
	FindUpcomingUserEvents(ctx context.Context, eventType string) ([]*UserEvent, int, error)
	CreateUserEvent(ctx context.Context, userEvent *UserEvent) error
	UpdateUserEvent(ctx context.Context, id int, update UserEventUpdate) (*UserEvent, error)
}
