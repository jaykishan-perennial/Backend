package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"license-management-backend/database"
	"license-management-backend/models"
)

func SDKSignup(c *gin.Context) {
	var req signupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		msg := parseValidationError(err)
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": msg})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	req.Phone = strings.TrimSpace(req.Phone)

	if len(req.Name) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Name must be at least 2 characters"})
		return
	}
	if len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Password must be at least 6 characters"})
		return
	}
	if len(req.Phone) < 7 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Phone must be at least 7 characters"})
		return
	}

	var existingUser models.User
	if err := database.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "Email already registered"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to create account"})
		return
	}

	tx := database.DB.Begin()

	user := models.User{Email: req.Email, PasswordHash: string(hash), Role: "customer"}
	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to create account"})
		return
	}

	apiKey := "sk-sdk-" + uuid.New().String()
	customer := models.Customer{UserID: user.ID, Name: req.Name, Phone: req.Phone, APIKey: apiKey}
	if err := tx.Create(&customer).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to create account"})
		return
	}

	tx.Commit()

	c.JSON(http.StatusCreated, gin.H{
		"success":    true,
		"api_key":    apiKey,
		"name":       customer.Name,
		"phone":      customer.Phone,
		"expires_in": 3600,
	})
}

func SDKLogin(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid input: " + err.Error()})
		return
	}

	var user models.User
	if err := database.DB.Where("email = ? AND role = ?", req.Email, "customer").First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Invalid credentials"})
		return
	}

	var customer models.Customer
	if err := database.DB.Where("user_id = ?", user.ID).First(&customer).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Customer profile not found"})
		return
	}

	apiKey := "sk-sdk-" + uuid.New().String()
	database.DB.Model(&customer).Update("api_key", apiKey)

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"api_key":    apiKey,
		"name":       customer.Name,
		"phone":      customer.Phone,
		"expires_in": 3600,
	})
}

func SDKListPacks(c *gin.Context) {
	search := strings.TrimSpace(c.Query("search"))

	query := database.DB.Model(&models.SubscriptionPack{})
	if search != "" {
		query = query.Where("name LIKE ? OR sku LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	var packs []models.SubscriptionPack
	query.Order("name ASC").Find(&packs)

	type packItem struct {
		ID             uint    `json:"id"`
		Name           string  `json:"name"`
		Description    string  `json:"description"`
		SKU            string  `json:"sku"`
		Price          float64 `json:"price"`
		ValidityMonths int     `json:"validity_months"`
	}

	result := make([]packItem, len(packs))
	for i, p := range packs {
		result[i] = packItem{
			ID:             p.ID,
			Name:           p.Name,
			Description:    p.Description,
			SKU:            p.SKU,
			Price:          p.Price,
			ValidityMonths: p.ValidityMonths,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

func SDKGetSubscription(c *gin.Context) {
	customerID, _ := c.Get("customer_id")

	var subscription models.Subscription
	if err := database.DB.Preload("Pack").
		Where("customer_id = ? AND status IN ?", customerID, []string{"active", "requested", "approved"}).
		Order("created_at desc").
		First(&subscription).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "No active subscription found"})
		return
	}

	isValid := subscription.Status == "active" &&
		subscription.ExpiresAt != nil &&
		subscription.ExpiresAt.After(time.Now())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"subscription": gin.H{
			"id":          subscription.ID,
			"pack_name":   subscription.Pack.Name,
			"pack_sku":    subscription.Pack.SKU,
			"price":       subscription.Pack.Price,
			"status":      subscription.Status,
			"assigned_at": subscription.AssignedAt,
			"expires_at":  subscription.ExpiresAt,
			"is_valid":    isValid,
		},
	})
}

func SDKRequestSubscription(c *gin.Context) {
	customerID, _ := c.Get("customer_id")

	var req struct {
		PackSKU string `json:"pack_sku" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid input: " + err.Error()})
		return
	}

	var pack models.SubscriptionPack
	if err := database.DB.Where("sku = ?", req.PackSKU).First(&pack).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Subscription pack not found"})
		return
	}

	var activeCount int64
	database.DB.Model(&models.Subscription{}).
		Where("customer_id = ? AND status IN ?", customerID, []string{"active", "requested", "approved"}).
		Count(&activeCount)
	if activeCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "You already have an active or pending subscription"})
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

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Subscription request submitted successfully",
		"subscription": gin.H{
			"id":           subscription.ID,
			"status":       subscription.Status,
			"requested_at": subscription.RequestedAt,
		},
	})
}

func SDKDeactivateSubscription(c *gin.Context) {
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
		"success":        true,
		"message":        "Subscription deactivated successfully",
		"deactivated_at": now,
	})
}

func SDKGetSubscriptionHistory(c *gin.Context) {
	customerID, _ := c.Get("customer_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	sort := c.DefaultQuery("sort", "desc")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	if sort != "asc" && sort != "desc" {
		sort = "desc"
	}
	offset := (page - 1) * limit

	var total int64
	database.DB.Model(&models.Subscription{}).Where("customer_id = ?", customerID).Count(&total)

	var subscriptions []models.Subscription
	database.DB.Preload("Pack").
		Where("customer_id = ?", customerID).
		Offset(offset).Limit(limit).
		Order("created_at " + sort).
		Find(&subscriptions)

	history := make([]gin.H, len(subscriptions))
	for i, sub := range subscriptions {
		history[i] = gin.H{
			"id":          sub.ID,
			"pack_name":   sub.Pack.Name,
			"pack_sku":    sub.Pack.SKU,
			"price":       sub.Pack.Price,
			"status":      sub.Status,
			"requested_at": sub.RequestedAt,
			"approved_at": sub.ApprovedAt,
			"assigned_at": sub.AssignedAt,
			"expires_at":  sub.ExpiresAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"history": history,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}
