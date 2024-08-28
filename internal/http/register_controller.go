package http

import (
	laundryNotify "jallier/laundry-notify"
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/gin-gonic/gin"
)

func (s *HttpServer) registerRegisterRoutes() {
	routerGroup := s.router.Group("/register")
	routerGroup.POST("", s.handleRegister)
	routerGroup.GET("", s.handleRegister)
}

type RegisterRequest struct {
	Name string `form:"name"`
	Type string `form:"type"`
}

func (s *HttpServer) handleRegister(c *gin.Context) {
	var req RegisterRequest
	c.Bind(&req)

	if req.Name == "" {
		c.HTML(http.StatusOK, "registered", gin.H{
			"error": "Name is required",
		})
		return
	}

	if req.Type == "" || (req.Type != laundryNotify.WASHER_EVENT && req.Type != laundryNotify.DRYER_EVENT) {
		c.HTML(http.StatusOK, "registered", gin.H{
			"error": "Valid type is required",
		})
		return
	}

	user, err := s.UserService.FindUserByName(s.ctx, req.Name)
	if err != nil {
		log.Error("Error finding user by name", "error", err)
	}
	if user == nil {
		log.Debug("User not found", "name", req.Name)
		user = &laundryNotify.User{Name: req.Name}
		err = s.UserService.CreateUser(s.ctx, user)
		if err != nil {
			log.Error("Error creating user", "error", err)
			c.HTML(http.StatusOK, "registered", gin.H{
				"error": "Error creating user",
			})
			return
		}
	} else {
		log.Debug("User found", "user", user)
	}
	log.Info("Registering user interest")

	mostRecentEvent, err := s.EventService.FindMostRecentEvent(s.ctx, req.Type)
	if err != nil {
		log.Error("Error finding most recent event", "error", err)
		c.HTML(http.StatusOK, "registered", gin.H{
			"error": "Error finding most recent event",
		})
		return
	}

	// If finished at isn't set, then this event is ongoing
	if mostRecentEvent != nil && mostRecentEvent.FinishedAt.Valid {
		log.Debug("Event is finished", "event", mostRecentEvent)
		// Event is finished
		// First check if they have already registered for the next event
		_, n, err := s.UserEventService.FindByUserName(s.ctx, user.Name, req.Type)
		if err != nil {
			log.Error("Error finding user events by name", "error", err)
			c.HTML(http.StatusOK, "registered", gin.H{
				"error": "Error finding user events by name",
			})
			return
		}
		log.Debug("User event count", "count", n)
		if n > 0 {
			log.Info("User already registered for next event", "user", user)
			c.HTML(http.StatusOK, "registered", gin.H{
				// "error": "User already registered for next event",
				"title": "Laundry Notify",
				"name":  user.Name,
			})
			return
		}
		// If they haven't, register them for the next event that is created
		userEvent := &laundryNotify.UserEvent{
			UserId: user.Id,
			Type:   req.Type,
		}
		err = s.UserEventService.CreateUserEvent(s.ctx, userEvent)
		if err != nil {
			log.Error("Error creating user event", "error", err)
			c.HTML(http.StatusOK, "registered", gin.H{
				"error": "Error creating user event",
			})
			return
		}
		log.Info("User registered for next event", "user", user, "event", mostRecentEvent)
	} else {
		log.Debug("Event is ongoing", "event", mostRecentEvent)
		// Event is ongoing
		// Check if they are already registered for this event
		_, n, err := s.UserEventService.FindByUserName(s.ctx, user.Name, req.Type)
		if err != nil {
			log.Error("Error finding user events by name", "error", err)
			c.HTML(http.StatusOK, "registered", gin.H{
				"error": "Error finding user events by name",
			})
			return
		}
		if n > 0 {
			log.Info("User already registered for this event", "user", user)
			c.HTML(http.StatusOK, "registered", gin.H{
				// "error": "User already registered for this event",
				"title":                "Laundry Notify",
				"name":                 user.Name,
				"previouslyRegistered": true,
				"ntfyBaseTopic":        s.Config.NtfyBaseTopic,
			})
			return
		}
		// If not, register for it
		userEvent := &laundryNotify.UserEvent{
			UserId:  user.Id,
			Type:    req.Type,
			EventId: mostRecentEvent.Id,
		}
		err = s.UserEventService.CreateUserEvent(s.ctx, userEvent)
		if err != nil {
			log.Error("Error creating user event", "error", err)
			c.HTML(http.StatusOK, "registered", gin.H{
				"error": "Error creating user event",
			})
			return
		}
		log.Info("User registered for ongoing event", "user", user, "event", mostRecentEvent)
		c.HTML(http.StatusOK, "registered", gin.H{
			"success": "User registered for ongoing event",
		})
	}
}
