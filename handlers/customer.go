package handlers

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"license-management-backend/database"
	"license-management-backend/models"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

type createCustomerRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password"`
	Phone    string `json:"phone"`
}

type updateCustomerRequest struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

func ListCustomers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	query := database.DB.Model(&models.Customer{})
	if search != "" {
		query = query.Joins("JOIN users ON users.id = customers.user_id").
			Where("customers.name LIKE ? OR users.email LIKE ? OR customers.phone LIKE ?",
				"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	query.Count(&total)

	var customers []models.Customer
	database.DB.Preload("User").Offset(offset).Limit(limit).Order("customers.created_at DESC")
	q := database.DB.Preload("User").Offset(offset).Limit(limit).Order("customers.created_at DESC")
	if search != "" {
		q = q.Joins("JOIN users ON users.id = customers.user_id").
			Where("customers.name LIKE ? OR users.email LIKE ? OR customers.phone LIKE ?",
				"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	q.Find(&customers)

	type customerResponse struct {
		ID        uint   `json:"id"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		Phone     string `json:"phone"`
		CreatedAt string `json:"created_at"`
	}

	result := make([]customerResponse, 0, len(customers))
	for _, cust := range customers {
		result = append(result, customerResponse{
			ID:        cust.ID,
			Name:      cust.Name,
			Email:     cust.User.Email,
			Phone:     cust.Phone,
			CreatedAt: cust.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

func CreateCustomer(c *gin.Context) {
	var req createCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid input: " + err.Error()})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	req.Phone = strings.TrimSpace(req.Phone)

	if len(req.Name) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Name must be at least 2 characters"})
		return
	}
	if !emailRegex.MatchString(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid email format"})
		return
	}
	if req.Phone != "" && len(req.Phone) < 7 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Phone number must be at least 7 characters"})
		return
	}

	password := req.Password
	if password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Password is required"})
		return
	}
	if len(password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Password must be at least 6 characters"})
		return
	}

	var existingUser models.User
	if err := database.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "Email already registered"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to hash password"})
		return
	}

	tx := database.DB.Begin()

	user := models.User{
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         "customer",
	}
	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to create user"})
		return
	}

	customer := models.Customer{
		UserID: user.ID,
		Name:   req.Name,
		Phone:  req.Phone,
	}
	if err := tx.Create(&customer).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to create customer"})
		return
	}

	tx.Commit()

	LogAudit(c, "create", "customer", customer.ID, "Created customer: "+customer.Name)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"id":    customer.ID,
			"name":  customer.Name,
			"email": user.Email,
			"phone": customer.Phone,
		},
	})
}

func GetCustomer(c *gin.Context) {
	id := c.Param("customer_id")

	var customer models.Customer
	if err := database.DB.Preload("User").First(&customer, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Customer not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":         customer.ID,
			"name":       customer.Name,
			"email":      customer.User.Email,
			"phone":      customer.Phone,
			"created_at": customer.CreatedAt,
		},
	})
}

func UpdateCustomer(c *gin.Context) {
	id := c.Param("customer_id")

	var customer models.Customer
	if err := database.DB.First(&customer, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Customer not found"})
		return
	}

	var req updateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid input: " + err.Error()})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Phone = strings.TrimSpace(req.Phone)

	if req.Name != "" && len(req.Name) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Name must be at least 2 characters"})
		return
	}
	if req.Phone != "" && len(req.Phone) < 7 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Phone number must be at least 7 characters"})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}

	if len(updates) > 0 {
		database.DB.Model(&customer).Updates(updates)
	}

	database.DB.Preload("User").First(&customer, id)

	LogAudit(c, "update", "customer", customer.ID, "Updated customer: "+customer.Name)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":    customer.ID,
			"name":  customer.Name,
			"email": customer.User.Email,
			"phone": customer.Phone,
		},
	})
}

func DeleteCustomer(c *gin.Context) {
	id := c.Param("customer_id")

	var customer models.Customer
	if err := database.DB.First(&customer, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Customer not found"})
		return
	}

	database.DB.Delete(&customer)

	LogAudit(c, "delete", "customer", customer.ID, "Deleted customer: "+customer.Name)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Customer deleted successfully",
	})
}
