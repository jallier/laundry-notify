package http

import (
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/gin-gonic/gin"
)

func (s *HttpServer) registerSearchRoute() {
	s.router.POST("/search", s.handleSearch)
}

type SearchRequest struct {
	Name string `form:"name"`
}

func (s *HttpServer) handleSearch(c *gin.Context) {
	var req SearchRequest
	c.Bind(&req)
	log.Debug("Received search request", "name", req.Name)

	users, n, err := s.UserService.FindMostRecentUsers(s.ctx, req.Name)
	if err != nil {
		log.Error("Error finding most recent users", "error", err)
	}
	log.Debug("Found users", "users", users, "count", n)

	c.HTML(http.StatusOK, "partials/search.html", gin.H{
		"users": users,
	})
}
