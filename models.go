package main

import (
	"time"

	"gorm.io/gorm"
)

type ctxKey struct{}

type User struct {
	ID       uint
	Email    string `gorm:"size:100;not null"`
	Password string `json:"-" gorm:"size:100;not null"`
	Salt     string `json:"-" gorm:"size:64;not null"`
	Admin    bool   `json:"-" gorm:"default:false"`
}

type Shop struct {
	ID          uint       `json:"id"`
	Name        string     `json:"name" gorm:"size:100;not null"`
	Description string     `json:"description" gorm:"not null"`
	User        User       `json:"-" gorm:"foreignkey:UserID;not null"`
	UserID      uint       `json:"-"`
	Locations   []Location `json:"-" gorm:"constraint:OnDelete:CASCADE;"`
	Products    []Product  `json:"-" gorm:"constraint:OnDelete:CASCADE;"`
}

type Product struct {
	ID          uint       `json:"id"`
	Name        string     `json:"name" gorm:"size:100;not null"`
	Description string     `json:"description" gorm:"not null"`
	ShopID      uint       `json:"-"`
	Categories  []Category `json:"-" gorm:"many2many:product_categories;"`
}

type Location struct {
	ID          uint   `json:"id"`
	Coordinates string `json:"coordinates" gorm:"size:100;not null"`
	ShopID      uint   `json:"-"`
}

type Category struct {
	ID   uint   `json:"id"`
	Name string `json:"name" gorm:"size:100;not null"`
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
