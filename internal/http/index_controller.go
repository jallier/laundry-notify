package http

import (
	laundryNotify "jallier/laundry-notify"
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
	mostRecentWasherEvent, err := s.EventService.FindMostRecentEvent(s.ctx, laundryNotify.WASHER_EVENT)
	if err != nil {
		log.Error("Error finding most recent event", "error", err)
	}
	mostRecentDryerEvent, err := s.EventService.FindMostRecentEvent(s.ctx, laundryNotify.DRYER_EVENT)
	if err != nil {
		log.Error("Error finding most recent event", "error", err)
	}

	c.HTML(http.StatusOK, "index", gin.H{
		"title":                 "Laundry Notify",
		"mostRecentWasherEvent": mostRecentWasherEvent,
		"mostRecentDryerEvent":  mostRecentDryerEvent,
		"user":                  user[0],
	})
}
