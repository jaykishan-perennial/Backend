package models

import (
	"time"

	"gorm.io/gorm"
)

type SubscriptionPack struct {
	ID             uint           `json:"id" gorm:"primaryKey"`
	Name           string         `json:"name" gorm:"not null"`
	Description    string         `json:"description"`
	SKU            string         `json:"sku" gorm:"uniqueIndex;not null"`
	Price          float64        `json:"price" gorm:"not null"`
	ValidityMonths int            `json:"validity_months" gorm:"not null"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}
