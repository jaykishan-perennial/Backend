package models

import "time"

type Subscription struct {
	ID            uint             `json:"id" gorm:"primaryKey"`
	CustomerID    uint             `json:"customer_id" gorm:"not null;index"`
	Customer      Customer         `json:"customer,omitempty" gorm:"foreignKey:CustomerID"`
	PackID        uint             `json:"pack_id" gorm:"not null;index"`
	Pack          SubscriptionPack `json:"pack,omitempty" gorm:"foreignKey:PackID"`
	Status        string           `json:"status" gorm:"not null;default:'requested';index"`
	RequestedAt   *time.Time       `json:"requested_at,omitempty"`
	ApprovedAt    *time.Time       `json:"approved_at,omitempty"`
	AssignedAt    *time.Time       `json:"assigned_at,omitempty"`
	ExpiresAt     *time.Time       `json:"expires_at,omitempty" gorm:"index"`
	DeactivatedAt *time.Time       `json:"deactivated_at,omitempty"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}
