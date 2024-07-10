package main

import (
	"database/sql"
	"net/http"
	"os"

	"github.com/charmbracelet/log"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/foolin/goview/supports/ginview"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

// What needs to happen:
// A user registers their interest with their name and is given the ntfy topic link
// the app subscribes to laundry events on mqtt
// Home assistant will publish messages to this topic
// If a user has registered interest, send the next 'finished' message to the ntfy topic + the users name
// Delete the user's "interest"

var db *sql.DB
var isDev bool

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Info("Error loading .env file. Either one not provided or running in prod mode")
	}
	env := getEnv("ENV")
	isDev = false
	if env == "dev" || env == "development" {
		log.SetLevel(log.DebugLevel)
		isDev = true
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
		created_at datetime
	);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}
	log.Debug("Database setup")

	opts := MQTT.NewClientOptions()
	opts.AddBroker("mqtt://10.0.0.3:1883")
	opts.SetClientID("laundry-notify")
	opts.SetUsername("")
	opts.SetPassword("")

	log.Info("starting subscriber")
	choke := make(chan [2]string)

	opts.SetDefaultPublishHandler(func(client MQTT.Client, msg MQTT.Message) {
		choke <- [2]string{msg.Topic(), string(msg.Payload())}
	})

	client := MQTT.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	log.Info("Connected")

	topic := "test"
	if token := client.Subscribe(topic, byte(0), nil); token.Wait() && token.Error() != nil {
		log.Error(token.Error())
		os.Exit(1)
	}
	log.Info("Subscribed to 'test' topic")

	for {
		incoming := <-choke
		log.Debug("RECEIVED TOPIC: %s MESSAGE: %s\n", "topic", incoming[0], "message", incoming[1])
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
		register.GET("/", func(c *gin.Context) {
			var query SearchRequest
			c.Bind(&query)

			c.HTML(http.StatusOK, "registered", gin.H{
				"title": "Register",
				"name":  query.Name,
			})
		})
	}

	router.Run()
}

// Simple ping function
func getPing(c *gin.Context) {
	c.String(http.StatusOK, "PONG")
}

func getRecentUsers() []string {
	rows, err := db.Query("select name from users order by created_at desc limit 1")
	if err != nil {
		log.Fatal("Error querying users table")
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

	return users
}

func handleIndex(c *gin.Context) {
	users := getRecentUsers()
	c.HTML(http.StatusOK, "index", gin.H{
		"title": "Laundry Notifications",
		"users": users[:1],
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
		users = getRecentUsers()
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
			// TODO: redirect to generic error page for these kinds of errors
			log.Fatal("Error adding user", "error", err)
		}
		log.Debug("User added", "name", name)
	}

	c.HTML(http.StatusOK, "registered", gin.H{
		"name": req.Name,
	})
}
