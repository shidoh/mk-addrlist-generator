package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"mk-addrlist-generator/pkg/config"
	"mk-addrlist-generator/pkg/generator"
)

type Server struct {
	config    *config.Config
	generator *generator.Generator
}

func NewServer(cfg *config.Config) *Server {
	return &Server{
		config:    cfg,
		generator: generator.NewGenerator(cfg),
	}
}

func (s *Server) SetupRoutes(r *gin.Engine) {
	r.GET("/lists/all", s.handleGetAllLists)
	r.GET("/list/:name", s.handleGetListByName)
}

func (s *Server) handleGetAllLists(c *gin.Context) {
	scripts, err := s.generator.GenerateAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := make(map[string]string)
	for name, script := range scripts {
		result[name] = script
	}

	c.Header("Content-Type", "text/plain")
	for _, script := range result {
		c.String(http.StatusOK, "%s\n", script)
	}
}

func (s *Server) handleGetListByName(c *gin.Context) {
	name := c.Param("name")
	
	list, exists := s.config.Lists[name]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("list %q not found", name)})
		return
	}

	script, err := s.generator.GenerateList(name, &list)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("error generating list %q: %v", name, err)})
		return
	}

	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, "%s\n", script)
}