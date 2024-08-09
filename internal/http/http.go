package http

import (
	"context"
	laundryNotify "jallier/laundry-notify"
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/foolin/goview/supports/ginview"
	"github.com/gin-gonic/gin"
)

type HttpServer struct {
	router *gin.Engine
	Config struct {
		Env string
	}
	UserService  laundryNotify.UserService
	EventService laundryNotify.EventService
	ctx          context.Context
	cancel       func()
}

func NewHttpServer() *HttpServer {
	server := &HttpServer{
		router: gin.Default(),
	}
	server.ctx, server.cancel = context.WithCancel(context.Background())

	server.router.HTMLRender = ginview.Default()
	server.router.Static("/static", "./static") // TODO: this should be moved into the same dir

	server.router.GET("/ping", handlePing)

	// Register controllers with router
	server.registerIndexRoute()
	registerRouterGroup := server.router.Group("/register")
	server.registerRegisterRoutes(registerRouterGroup)

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
