package http

import (
	"context"
	"fmt"
	laundryNotify "jallier/laundry-notify"
	"net/http"
	"text/template"

	"github.com/charmbracelet/log"
	"github.com/foolin/goview"
	"github.com/foolin/goview/supports/ginview"
	"github.com/gin-gonic/gin"
)

type HttpServer struct {
	router *gin.Engine
	Config struct {
		Env string
	}
	UserService      laundryNotify.UserService
	EventService     laundryNotify.EventService
	UserEventService laundryNotify.UserEventService
	ctx              context.Context
	cancel           func()
}

func NewHttpServer() *HttpServer {
	server := &HttpServer{
		router: gin.Default(),
	}
	server.ctx, server.cancel = context.WithCancel(context.Background())

	renderer := ginview.New(goview.Config{
		Root:      "views",
		Extension: ".html",
		Master:    "layouts/master",
		Funcs: template.FuncMap{
			"dict": dict,
		},
	})
	server.router.HTMLRender = renderer
	server.router.Static("/static", "./static") // TODO: this should be moved into the same dir

	server.router.GET("/ping", handlePing)

	// Register controllers with router
	server.registerIndexRoute()
	server.registerSearchRoute()
	server.registerRegisterRoutes()

	return server
}

func (s *HttpServer) Open() {
	if s.Config.Env == "dev" || s.Config.Env == "development" {
		log.Debug("running http server in development mode")
		s.router.ForwardedByClientIP = true
		s.router.SetTrustedProxies([]string{"127.0.0.1"})
	}

	go func() {
		err := s.router.Run(":8080")
		if err != nil {
			log.Error("Error starting HTTP server", "error", err)
			s.cancel()
		}
	}()

	log.Info("HTTP server started")
}

func handlePing(c *gin.Context) {
	c.String(http.StatusOK, "pong")
}

func dict(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, fmt.Errorf("invalid dict call")
	}
	dict := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict keys must be strings")
		}
		dict[key] = values[i+1]
	}
	return dict, nil
}
