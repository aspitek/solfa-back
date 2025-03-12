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
	IsAdmin		      bool   `json:"is_admin" gorm:"default:true"`
	IsVerified        bool   `json:"is_verified" gorm:"default:true"`
	VerificationToken string `json:"-"`
}

