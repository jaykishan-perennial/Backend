package main

import (
	"log"
	"net"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"license-management-backend/config"
	"license-management-backend/database"
	"license-management-backend/middleware"
	"license-management-backend/routes"
)

func main() {
	cfg := config.Load()

	database.Init(cfg)

	go startExpiredSubscriptionChecker()

	r := gin.Default()

	r.Use(middleware.RateLimiter(10, 20))

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-API-Key"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	routes.Setup(r, cfg)

	r.Static("/web", "./web")
	r.GET("/admin", func(c *gin.Context) {
		c.File("./web/index.html")
	})

	addr := "0.0.0.0:" + cfg.Port
	log.Printf("Server starting on %s (IPv4)", addr)
	listener, err := net.Listen("tcp4", addr)
	if err != nil {
		log.Fatal("Failed to create listener:", err)
	}
	if err := r.RunListener(listener); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func startExpiredSubscriptionChecker() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	check := func() {
		result := database.DB.Exec(
			`UPDATE subscriptions SET status = 'expired', updated_at = ? WHERE status = 'active' AND expires_at IS NOT NULL AND expires_at < ?`,
			time.Now(), time.Now(),
		)
		if result.RowsAffected > 0 {
			log.Printf("Expired %d subscription(s)", result.RowsAffected)
		}
	}

	check()
	for range ticker.C {
		check()
	}
}
