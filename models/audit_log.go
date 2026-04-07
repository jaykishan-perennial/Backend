package models

import "time"

type AuditLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"index"`
	Action    string    `json:"action" gorm:"not null;index"`
	Entity    string    `json:"entity" gorm:"not null;index"`
	EntityID  uint      `json:"entity_id"`
	Details   string    `json:"details"`
	IPAddress string    `json:"ip_address"`
	CreatedAt time.Time `json:"created_at" gorm:"index"`
}
