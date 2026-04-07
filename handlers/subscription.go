package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"license-management-backend/database"
	"license-management-backend/models"
)

func ListSubscriptions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	status := c.Query("status")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	query := database.DB.Model(&models.Subscription{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var subscriptions []models.Subscription
	q := database.DB.Preload("Customer").Preload("Pack").Offset(offset).Limit(limit).Order("created_at DESC")
	if status != "" {
		q = q.Where("status = ?", status)
	}
	q.Find(&subscriptions)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    subscriptions,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

func ApproveSubscription(c *gin.Context) {
	subID := c.Param("subscription_id")

	var subscription models.Subscription
	if err := database.DB.First(&subscription, subID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Subscription not found"})
		return
	}

	if subscription.Status != "requested" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Only requested subscriptions can be approved"})
		return
	}

	now := time.Now()
	subscription.Status = "approved"
	subscription.ApprovedAt = &now
	database.DB.Save(&subscription)

	database.DB.Preload("Customer").Preload("Pack").First(&subscription, subID)

	LogAudit(c, "approve", "subscription", subscription.ID, "Approved subscription for customer "+subscription.Customer.Name)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    subscription,
		"message": "Subscription approved successfully",
	})
}

type assignRequest struct {
	PackID uint `json:"pack_id" binding:"required"`
}

func AssignSubscription(c *gin.Context) {
	customerID := c.Param("customer_id")

	var customer models.Customer
	if err := database.DB.First(&customer, customerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Customer not found"})
		return
	}

	var req assignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid input: " + err.Error()})
		return
	}

	var pack models.SubscriptionPack
	if err := database.DB.First(&pack, req.PackID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Subscription pack not found"})
		return
	}

	var activeCount int64
	database.DB.Model(&models.Subscription{}).
		Where("customer_id = ? AND status = ?", customer.ID, "active").
		Count(&activeCount)
	if activeCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "Customer already has an active subscription"})
		return
	}

	now := time.Now()
	expiresAt := now.AddDate(0, pack.ValidityMonths, 0)

	var existingApproved models.Subscription
	err := database.DB.Where("customer_id = ? AND pack_id = ? AND status = ?", customer.ID, pack.ID, "approved").
		First(&existingApproved).Error

	if err == nil {
		existingApproved.Status = "active"
		existingApproved.AssignedAt = &now
		existingApproved.ExpiresAt = &expiresAt
		database.DB.Save(&existingApproved)
		database.DB.Preload("Customer").Preload("Pack").First(&existingApproved, existingApproved.ID)
		LogAudit(c, "assign", "subscription", existingApproved.ID, "Assigned subscription to "+customer.Name)
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    existingApproved,
			"message": "Subscription assigned successfully",
		})
		return
	}

	subscription := models.Subscription{
		CustomerID:  customer.ID,
		PackID:      pack.ID,
		Status:      "active",
		RequestedAt: &now,
		AssignedAt:  &now,
		ExpiresAt:   &expiresAt,
	}

	if err := database.DB.Create(&subscription).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to assign subscription"})
		return
	}

	database.DB.Preload("Customer").Preload("Pack").First(&subscription, subscription.ID)

	LogAudit(c, "assign", "subscription", subscription.ID, "Directly assigned subscription to "+customer.Name)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    subscription,
		"message": "Subscription assigned successfully",
	})
}

func UnassignSubscription(c *gin.Context) {
	customerID := c.Param("customer_id")
	subID := c.Param("subscription_id")

	var subscription models.Subscription
	if err := database.DB.Where("id = ? AND customer_id = ?", subID, customerID).First(&subscription).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Subscription not found"})
		return
	}

	if subscription.Status != "active" && subscription.Status != "approved" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Only active or approved subscriptions can be unassigned"})
		return
	}

	now := time.Now()
	subscription.Status = "inactive"
	subscription.DeactivatedAt = &now
	database.DB.Save(&subscription)

	LogAudit(c, "unassign", "subscription", subscription.ID, "Unassigned subscription")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Subscription unassigned successfully",
	})
}

func ListSubscriptionPacks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	var total int64
	database.DB.Model(&models.SubscriptionPack{}).Count(&total)

	var packs []models.SubscriptionPack
	database.DB.Offset(offset).Limit(limit).Order("created_at DESC").Find(&packs)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    packs,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

type packRequest struct {
	Name           string  `json:"name" binding:"required"`
	Description    string  `json:"description"`
	SKU            string  `json:"sku" binding:"required"`
	Price          float64 `json:"price" binding:"required"`
	ValidityMonths int     `json:"validity_months" binding:"required,min=1,max=12"`
}

type updatePackRequest struct {
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	SKU            string  `json:"sku"`
	Price          float64 `json:"price"`
	ValidityMonths int     `json:"validity_months"`
}

func CreateSubscriptionPack(c *gin.Context) {
	var req packRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid input: " + err.Error()})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.SKU = strings.TrimSpace(req.SKU)
	req.Description = strings.TrimSpace(req.Description)

	if len(req.Name) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Pack name must be at least 2 characters"})
		return
	}
	if len(req.SKU) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "SKU must be at least 2 characters"})
		return
	}
	if req.Price <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Price must be greater than 0"})
		return
	}
	if req.ValidityMonths < 1 || req.ValidityMonths > 12 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Validity must be between 1 and 12 months"})
		return
	}

	var existing models.SubscriptionPack
	if err := database.DB.Where("sku = ?", req.SKU).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "SKU already exists"})
		return
	}

	pack := models.SubscriptionPack{
		Name:           req.Name,
		Description:    req.Description,
		SKU:            req.SKU,
		Price:          req.Price,
		ValidityMonths: req.ValidityMonths,
	}

	if err := database.DB.Create(&pack).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to create subscription pack"})
		return
	}

	LogAudit(c, "create", "subscription_pack", pack.ID, "Created pack: "+pack.Name)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    pack,
	})
}

func UpdateSubscriptionPack(c *gin.Context) {
	id := c.Param("pack_id")

	var pack models.SubscriptionPack
	if err := database.DB.First(&pack, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Subscription pack not found"})
		return
	}

	var req updatePackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid input: " + err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.SKU != "" {
		updates["sku"] = req.SKU
	}
	if req.Price > 0 {
		updates["price"] = req.Price
	}
	if req.ValidityMonths >= 1 && req.ValidityMonths <= 12 {
		updates["validity_months"] = req.ValidityMonths
	}

	if len(updates) > 0 {
		database.DB.Model(&pack).Updates(updates)
	}

	database.DB.First(&pack, id)

	LogAudit(c, "update", "subscription_pack", pack.ID, "Updated pack: "+pack.Name)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    pack,
	})
}

func DeleteSubscriptionPack(c *gin.Context) {
	id := c.Param("pack_id")

	var pack models.SubscriptionPack
	if err := database.DB.First(&pack, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Subscription pack not found"})
		return
	}

	database.DB.Delete(&pack)

	LogAudit(c, "delete", "subscription_pack", pack.ID, "Deleted pack: "+pack.Name)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Subscription pack deleted successfully",
	})
}

func GetCustomerSubscription(c *gin.Context) {
	customerID, _ := c.Get("customer_id")

	var subscription models.Subscription
	if err := database.DB.Preload("Pack").
		Where("customer_id = ? AND status = ?", customerID, "active").
		First(&subscription).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "No active subscription found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    subscription,
	})
}

func RequestSubscription(c *gin.Context) {
	customerID, _ := c.Get("customer_id")

	var req struct {
		SKU string `json:"sku" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid input: " + err.Error()})
		return
	}

	var pack models.SubscriptionPack
	if err := database.DB.Where("sku = ?", req.SKU).First(&pack).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Subscription pack not found"})
		return
	}

	var activeCount int64
	database.DB.Model(&models.Subscription{}).
		Where("customer_id = ? AND status = ?", customerID, "active").
		Count(&activeCount)
	if activeCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "You already have an active subscription"})
		return
	}

	now := time.Now()
	subscription := models.Subscription{
		CustomerID:  customerID.(uint),
		PackID:      pack.ID,
		Status:      "requested",
		RequestedAt: &now,
	}

	if err := database.DB.Create(&subscription).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to request subscription"})
		return
	}

	database.DB.Preload("Pack").First(&subscription, subscription.ID)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    subscription,
		"message": "Subscription requested successfully",
	})
}

func DeactivateSubscription(c *gin.Context) {
	customerID, _ := c.Get("customer_id")

	var subscription models.Subscription
	if err := database.DB.Where("customer_id = ? AND status = ?", customerID, "active").
		First(&subscription).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "No active subscription found"})
		return
	}

	now := time.Now()
	subscription.Status = "inactive"
	subscription.DeactivatedAt = &now
	database.DB.Save(&subscription)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Subscription deactivated successfully",
	})
}

func GetSubscriptionHistory(c *gin.Context) {
	customerID, _ := c.Get("customer_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	sortBy := c.DefaultQuery("sort", "created_at")
	order := c.DefaultQuery("order", "desc")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	allowedSorts := map[string]bool{"created_at": true, "status": true, "expires_at": true}
	if !allowedSorts[sortBy] {
		sortBy = "created_at"
	}
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	var total int64
	database.DB.Model(&models.Subscription{}).Where("customer_id = ?", customerID).Count(&total)

	var subscriptions []models.Subscription
	database.DB.Preload("Pack").
		Where("customer_id = ?", customerID).
		Offset(offset).Limit(limit).
		Order(sortBy + " " + order).
		Find(&subscriptions)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    subscriptions,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}
