package http

import (
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/gin-gonic/gin"
)

func (s *HttpServer) registerIndexRoute() {
	s.router.GET("/", s.handleIndex)
}

func (s *HttpServer) handleIndex(c *gin.Context) {
	user, _, err := s.UserService.FindMostRecentUsers(s.ctx, "")
	if err != nil {
		log.Error("Error finding most recent user", "error", err)
	}
	mostRecentEvent, err := s.EventService.FindMostRecentEvent(s.ctx, "")
	if err != nil {
		log.Error("Error finding most recent event", "error", err)
	}
	c.HTML(http.StatusOK, "index", gin.H{
		"title":           "Laundry Notify",
		"mostRecentEvent": mostRecentEvent,
		"user":            user[0],
	})
}
