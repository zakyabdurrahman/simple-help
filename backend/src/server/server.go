package server

import (
	"database/sql"
	"time"

	"simplehelp/src/controllers"
	"simplehelp/src/middleware"
	"simplehelp/src/models"
	"simplehelp/src/repositories"
	"simplehelp/src/services"

	"github.com/gin-gonic/gin"
)

// New builds the configured Gin engine with all routes registered.
func New(db *sql.DB, jwtSecret string, jwtExpiry time.Duration) *gin.Engine {
	r := gin.Default()

	// Dependency wiring: repository -> service -> controller.
	userRepo := repositories.NewUserRepository(db)
	ticketRepo := repositories.NewTicketRepository(db)
	commentRepo := repositories.NewCommentRepository(db)

	authService := services.NewAuthService(userRepo, jwtSecret, jwtExpiry)
	ticketService := services.NewTicketService(ticketRepo, commentRepo)
	userService := services.NewUserService(userRepo)

	authCtrl := controllers.NewAuthController(authService)
	ticketCtrl := controllers.NewTicketController(ticketService, userRepo)
	userCtrl := controllers.NewUserController(userService)

	api := r.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", authCtrl.Register)
			auth.POST("/login", authCtrl.Login)
		}

		// Authenticated routes.
		authed := api.Group("")
		authed.Use(middleware.AuthRequired(jwtSecret))
		{
			authed.GET("/me", authCtrl.Me)

			// Tickets.
			authed.POST("/tickets", ticketCtrl.Create)
			authed.GET("/tickets", ticketCtrl.List)
			authed.GET("/tickets/:id", ticketCtrl.Get)
			authed.PATCH("/tickets/:id", ticketCtrl.Update)
			authed.POST("/tickets/:id/comments", ticketCtrl.CreateComment)

			// Admin-only user management.
			admin := authed.Group("")
			admin.Use(middleware.RequireRole(models.RoleAdmin))
			{
				admin.GET("/users", userCtrl.List)
				admin.PATCH("/users/:id/role", userCtrl.UpdateRole)
			}
		}
	}

	// Liveness probe.
	r.GET("/healthz", func(c *gin.Context) { c.String(200, "ok") })

	return r
}
