package api

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

func (s *Server) ServeStaticFiles() {
	s.router.GET("/", func(c *gin.Context) {
		c.File("public/index.html")
	})
	
	s.router.StaticFS("/static", http.Dir("public"))
}