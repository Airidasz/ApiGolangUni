package main

import (
	"time"

	"gorm.io/gorm"
)

type ctxKey struct{}

type User struct {
	ID       uint
	Name     string `json:"name" gorm:"size:100"`
	Email    string `gorm:"size:100;not null"`
	Password string `json:"-" gorm:"size:100;not null"`
	Salt     string `json:"-" gorm:"size:64;not null"`
	Admin    bool   `json:"-" gorm:"default:false"`
}

type Shop struct {
	ID          uint       `json:"-"`
	Name        string     `json:"name" gorm:"size:100;not null"`
	Description string     `json:"description" gorm:"not null"`
	User        User       `json:"-" gorm:"foreignkey:UserID;not null"`
	UserID      uint       `json:"-"`
	Locations   []Location `json:"-" gorm:"constraint:OnDelete:CASCADE;"`
	Products    []Product  `json:"-" gorm:"constraint:OnDelete:CASCADE;"`
}

type Product struct {
	ID          uint       `json:"-"`
	Name        string     `json:"name" gorm:"size:100;not null"`
	Description string     `json:"description" gorm:"not null"`
	ShopID      uint       `json:"-"`
	Categories  []Category `json:"categories" gorm:"many2many:product_categories;constraint:OnDelete:CASCADE;"`
}

type Location struct {
	ID     uint    `json:"-"`
	Type   string  `json:"type"  gorm:"size:30;not null"`
	Lat    float32 `json:"lat"  gorm:"not null"`
	Lng    float32 `json:"lng"  gorm:"not null"`
	ShopID uint    `json:"-"`
}

type Category struct {
	ID       uint      `json:"-"`
	Name     string    `json:"name" gorm:"size:100;not null"`
	File     string    `json:"file" gorm:"size:500"`
	Products []Product `json:"-" gorm:"many2many:product_categories;constraint:OnDelete:CASCADE;"`
}

type RefreshToken struct {
	Token     string `gorm:"primaryKey"`
	Email     string `gorm:"not null"`
	CreatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type ErrorJSON struct {
	Message string `json:"message"`
}
