package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *HttpServer) registerIndexRoute() {
	s.router.GET("/", s.handleIndex)
}

func (s *HttpServer) handleIndex(c *gin.Context) {
	c.String(http.StatusOK, "home")
}
