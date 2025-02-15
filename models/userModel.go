package models

import (
	"gorm.io/gorm"
)

// User représente un utilisateur de l'application
type User struct {
	gorm.Model
	Username          string `json:"username"`
	Email             string `json:"email" gorm:"unique"`
	Password          string `json:"-"`
	IsVerified        bool   `json:"is_verified" gorm:"default:false"`
	VerificationToken string `json:"-"`
}

