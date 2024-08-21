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
	Type string `form:"type"`
}

func (s *HttpServer) handleSearch(c *gin.Context) {
	var req SearchRequest
	c.Bind(&req)

	users, _, err := s.UserService.FindMostRecentUsers(s.ctx, req.Name)
	if err != nil {
		log.Error("Error finding most recent users", "error", err)
	}

	// Stupid goview workaround. Or maybe stupid me :thinking:
	templateName := "partials/search-washer.html"
	if req.Type == "dryer" {
		templateName = "partials/search-dryer.html"
	}

	c.HTML(http.StatusOK, templateName, gin.H{
		"users": users,
	})
}
