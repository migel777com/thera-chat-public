package server

import (
	"chatgpt/ai"
	"chatgpt/api/middleware"
	"chatgpt/config"
	"chatgpt/models"
	"context"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"time"
)

type Server struct {
	Configuration *config.Config
	Router        *gin.Engine
	Db            models.DbClient
	Cache         models.CacheClient
	AI            *ai.AI
}

func NewApiServer(config *config.Config, db models.DbClient, cache models.CacheClient, ai *ai.AI) *Server {
	return &Server{
		Configuration: config,
		Router:        gin.Default(),
		Db:            db,
		Cache:         cache,
		AI:            ai,
	}
}

func (s *Server) Init(ctx context.Context) {
	s.Router.Use(cors.New(cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		AllowOrigins:     []string{"*"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}),
		middleware.ErrorHandler())

	s.Router.GET("/", s.HealthCheck)
	s.Router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

func (s *Server) HealthCheck(c *gin.Context) {
	c.JSON(200, "OK")
	return
}
