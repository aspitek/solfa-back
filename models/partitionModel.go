package models

import "time"

type Partition struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Title       string    `json:"title"`
	Composer    string    `json:"composer"`
	Genre       string    `json:"genre"`
	Category    string    `json:"category"`
	ReleaseDate time.Time `json:"release_date"`
	Path        string    `json:"path"`
	Status      string    `json:"status"`  // "staging" ou "validated"
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
