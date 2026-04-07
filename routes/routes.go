package routes

import (
	"github.com/gin-gonic/gin"

	"license-management-backend/config"
	"license-management-backend/handlers"
	"license-management-backend/middleware"
)

func Setup(r *gin.Engine, cfg *config.Config) {
	authHandler := handlers.NewAuthHandler(cfg)

	api := r.Group("/api")
	{
		api.POST("/admin/login", authHandler.AdminLogin)
		api.POST("/customer/login", authHandler.CustomerLogin)
		api.POST("/customer/signup", authHandler.CustomerSignup)
	}

	adminV1 := r.Group("/api/v1/admin")
	adminV1.Use(middleware.JWTAuth(cfg, "admin"))
	{
		adminV1.GET("/dashboard", handlers.GetDashboard)

		adminV1.GET("/customers", handlers.ListCustomers)
		adminV1.POST("/customers", handlers.CreateCustomer)
		adminV1.GET("/customers/:customer_id", handlers.GetCustomer)
		adminV1.PUT("/customers/:customer_id", handlers.UpdateCustomer)
		adminV1.DELETE("/customers/:customer_id", handlers.DeleteCustomer)

		adminV1.GET("/subscription-packs", handlers.ListSubscriptionPacks)
		adminV1.POST("/subscription-packs", handlers.CreateSubscriptionPack)
		adminV1.PUT("/subscription-packs/:pack_id", handlers.UpdateSubscriptionPack)
		adminV1.DELETE("/subscription-packs/:pack_id", handlers.DeleteSubscriptionPack)

		adminV1.GET("/subscriptions", handlers.ListSubscriptions)
		adminV1.POST("/subscriptions/:subscription_id/approve", handlers.ApproveSubscription)
		adminV1.POST("/customers/:customer_id/assign-subscription", handlers.AssignSubscription)
		adminV1.DELETE("/customers/:customer_id/subscription/:subscription_id", handlers.UnassignSubscription)

		adminV1.GET("/audit-logs", handlers.ListAuditLogs)
	}

	customerV1 := r.Group("/api/v1/customer")
	customerV1.Use(middleware.JWTAuth(cfg, "customer"))
	{
		customerV1.GET("/subscription", handlers.GetCustomerSubscription)
		customerV1.POST("/subscription", handlers.RequestSubscription)
		customerV1.DELETE("/subscription", handlers.DeactivateSubscription)
		customerV1.GET("/subscription-history", handlers.GetSubscriptionHistory)
	}

	sdk := r.Group("/sdk")
	{
		sdk.POST("/auth/login", handlers.SDKLogin)
		sdk.POST("/auth/signup", handlers.SDKSignup)
	}

	sdkV1 := r.Group("/sdk/v1")
	sdkV1.Use(middleware.APIKeyAuth())
	{
		sdkV1.GET("/subscription-packs", handlers.SDKListPacks)
		sdkV1.GET("/subscription", handlers.SDKGetSubscription)
		sdkV1.POST("/subscription", handlers.SDKRequestSubscription)
		sdkV1.DELETE("/subscription", handlers.SDKDeactivateSubscription)
		sdkV1.GET("/subscription-history", handlers.SDKGetSubscriptionHistory)
	}
}
