package main

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ctxKey struct{}

type User struct {
	ID           uuid.UUID `gorm:"primary_key;type:uuid;default:uuid_generate_v4()"`
	Name         string    `json:"name" gorm:"size:100"`
	Email        string    `gorm:"size:100;not null"`
	Password     string    `json:"-" gorm:"size:100;not null"`
	Salt         string    `json:"-" gorm:"size:64;not null"`
	Permissions  string    `json:"-" gorm:"size:20"`
	ShopCodename *string   `json:"-"`
}

type Shop struct {
	ID          uuid.UUID  `json:"-" gorm:"primary_key;type:uuid;default:uuid_generate_v4()"`
	Name        *string    `json:"name" gorm:"size:100;not null"`
	Codename    string     `json:"codename" gorm:"size:100;not null"`
	Description *string    `json:"description" gorm:"not null"`
	User        User       `json:"-" gorm:"not null"`
	UserID      uuid.UUID  `json:"-"`
	Locations   []Location `json:"-" gorm:"constraint:OnDelete:CASCADE;"`
	Products    []Product  `json:"-" gorm:"constraint:OnDelete:CASCADE;"`
}

type Product struct {
	ID          uuid.UUID  `gorm:"primary_key;type:uuid;default:uuid_generate_v4()"`
	Name        string     `json:"name" gorm:"size:100;not null"`
	Codename    string     `json:"codename" gorm:"size:100;not null"`
	Description string     `json:"description" gorm:"not null"`
	Image       string     `json:"image" gorm:"size:500"`
	Shop        Shop       `json:"-" gorm:"not null"`
	ShopID      uuid.UUID  `json:"-"`
	Categories  []Category `json:"categories" gorm:"many2many:product_categories;constraint:OnDelete:CASCADE;"`
}

type Location struct {
	ID     uuid.UUID `gorm:"primary_key;type:uuid;default:uuid_generate_v4()"`
	Type   string    `json:"type"  gorm:"size:30;not null"`
	Lat    float32   `json:"lat"  gorm:"not null"`
	Lng    float32   `json:"lng"  gorm:"not null"`
	Shop   Shop      `json:"-" gorm:"not null"`
	ShopID uuid.UUID `json:"-"`
}

type Category struct {
	ID       uuid.UUID `json:"id" gorm:"primary_key;type:uuid;default:uuid_generate_v4()"`
	Name     *string   `json:"name" gorm:"size:100;not null"`
	Codename string    `json:"codename" gorm:"size:100;not null"`
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
