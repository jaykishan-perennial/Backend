package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"license-management-backend/database"
	"license-management-backend/models"
)

func LogAudit(c *gin.Context, action, entity string, entityID uint, details string) {
	var userID uint
	if v, ok := c.Get("user_id"); ok {
		if id, ok := v.(uint); ok {
			userID = id
		}
	}

	entry := models.AuditLog{
		UserID:    userID,
		Action:    action,
		Entity:    entity,
		EntityID:  entityID,
		Details:   details,
		IPAddress: c.ClientIP(),
	}
	if err := database.DB.Create(&entry).Error; err != nil {
		log.Printf("audit log insert failed: %v", err)
	}
}

func ListAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	var total int64
	database.DB.Model(&models.AuditLog{}).Count(&total)

	var logs []models.AuditLog
	database.DB.Offset(offset).Limit(limit).Order("created_at DESC").Find(&logs)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    logs,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}
