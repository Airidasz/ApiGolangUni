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
}

type Shop struct {
	ID          uint       `json:"id"`
	Name        string     `json:"name" gorm:"size:100;not null"`
	Description string     `json:"description" gorm:"not null"`
	User        User       `json:"-" gorm:"foreignkey:UserID;not null"`
	UserID      uint       `json:"-"`
	Locations   []Location `json:"-"`
	Products    []Product  `json:"-"`
}

type Product struct {
	ID          uint        `json:"id"`
	Name        string      `json:"name" gorm:"size:100;not null"`
	Description string      `json:"description" gorm:"not null"`
	ShopID      uint        `json:"-"`
	Categories  []*Category `json:"-" gorm:"many2many:product_categories;"`
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
