package server

import (
	"database/sql"

	"simplehelp/src/controllers"
	"simplehelp/src/repositories"
	"simplehelp/src/services"

	"github.com/gin-gonic/gin"
)

// New builds the configured Gin engine with all routes registered.
func New(db *sql.DB) *gin.Engine {
	r := gin.Default()

	// Dependency wiring: repository -> service -> controller.
	userRepo := repositories.NewUserRepository(db)
	authService := services.NewAuthService(userRepo)
	authCtrl := controllers.NewAuthController(authService)

	api := r.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", authCtrl.Register)
		}
	}

	// Liveness probe.
	r.GET("/healthz", func(c *gin.Context) { c.String(200, "ok") })

	return r
}
