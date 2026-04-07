package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"license-management-backend/database"
	"license-management-backend/models"
)

func GetDashboard(c *gin.Context) {
	var totalCustomers int64
	database.DB.Model(&models.Customer{}).Count(&totalCustomers)

	var activeSubscriptions int64
	database.DB.Model(&models.Subscription{}).Where("status = ?", "active").Count(&activeSubscriptions)

	var pendingRequests int64
	database.DB.Model(&models.Subscription{}).Where("status = ?", "requested").Count(&pendingRequests)

	var totalRevenue float64
	database.DB.Model(&models.Subscription{}).
		Select("COALESCE(SUM(subscription_packs.price), 0)").
		Joins("JOIN subscription_packs ON subscription_packs.id = subscriptions.pack_id").
		Where("subscriptions.status IN ?", []string{"active", "expired", "inactive"}).
		Scan(&totalRevenue)

	var recentActivities []models.Subscription
	database.DB.Preload("Customer").Preload("Pack").
		Order("updated_at DESC").Limit(10).Find(&recentActivities)

	type activity struct {
		ID           uint      `json:"id"`
		CustomerName string    `json:"customer_name"`
		PackName     string    `json:"pack_name"`
		Status       string    `json:"status"`
		Date         time.Time `json:"date"`
	}

	activities := make([]activity, 0, len(recentActivities))
	for _, s := range recentActivities {
		activities = append(activities, activity{
			ID:           s.ID,
			CustomerName: s.Customer.Name,
			PackName:     s.Pack.Name,
			Status:       s.Status,
			Date:         s.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"total_customers":      totalCustomers,
			"active_subscriptions": activeSubscriptions,
			"pending_requests":     pendingRequests,
			"total_revenue":        totalRevenue,
			"recent_activities":    activities,
		},
	})
}
