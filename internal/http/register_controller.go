package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *HttpServer) registerRegisterRoutes(routerGroup *gin.RouterGroup) {
	routerGroup.POST("", s.handleRegister)
	routerGroup.GET("", s.handleRegister)
}

func (s *HttpServer) handleRegister(c *gin.Context) {
	c.String(http.StatusOK, "register")
}
