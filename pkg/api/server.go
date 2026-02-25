package api

import (
	"fmt"
	"mk-addrlist-generator/pkg/config"
	"mk-addrlist-generator/pkg/generator"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Server struct {
	cfg       *config.Config
	generator *generator.Generator
	router    *gin.Engine
	server    *http.Server
}

func NewServer(cfg *config.Config) *Server {
	s := &Server{
		cfg:       cfg,
		generator: generator.NewGenerator(cfg),
		router:    gin.Default(),
	}

	// Register routes
	s.router.GET("/lists/all", s.HandleGetAllLists)
	s.router.GET("/list/:name", s.HandleGetListByName)

	return s
}

func (s *Server) Start(addr string) error {
	s.server = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}
	return s.server.ListenAndServe()
}

func (s *Server) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

func (s *Server) HandleGetAllLists(c *gin.Context) {
	script, err := s.generator.GenerateAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, script)
}

func (s *Server) HandleGetListByName(c *gin.Context) {
	name := c.Param("name")
	list, exists := s.cfg.Lists[name]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("list %s not found", name)})
		return
	}

	script, err := s.generator.GenerateList(name, list)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, script)
}
