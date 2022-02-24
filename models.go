package main

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ctxKey struct{}

type User struct {
	Base
	Name     string `json:"name" gorm:"size:100"`
	Email    string `gorm:"size:100;not null"`
	Password string `json:"-" gorm:"size:100;not null"`
	Salt     string `json:"-" gorm:"size:64;not null"`
	Admin    bool   `json:"-" gorm:"default:false"`
}

type Shop struct {
	Base
	Name        *string    `json:"name" gorm:"size:100;not null"`
	Description *string    `json:"description" gorm:"not null"`
	User        User       `json:"-" gorm:"foreignkey:UserID;not null"`
	UserID      uuid.UUID  `json:"-"`
	Locations   []Location `json:"-" gorm:"constraint:OnDelete:CASCADE;"`
	Products    []Product  `json:"-" gorm:"constraint:OnDelete:CASCADE;"`
}

type Product struct {
	Base
	Name        string     `json:"name" gorm:"size:100;not null"`
	Description string     `json:"description" gorm:"not null"`
	Shop        Shop       `json:"-" gorm:"foreignkey:ShopID;not null"`
	ShopID      uuid.UUID  `json:"-"`
	Categories  []Category `json:"categories" gorm:"many2many:product_categories;constraint:OnDelete:CASCADE;"`
}

type Location struct {
	Base
	Type   string    `json:"type"  gorm:"size:30;not null"`
	Lat    float32   `json:"lat"  gorm:"not null"`
	Lng    float32   `json:"lng"  gorm:"not null"`
	Shop   Shop      `json:"-" gorm:"foreignkey:ShopID;not null"`
	ShopID uuid.UUID `json:"-"`
}

type Category struct {
	Base
	Name     *string   `json:"name" gorm:"size:100;not null"`
	File     string   `json:"file" gorm:"size:500"`
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

func (base *Base) BeforeCreate(tx *gorm.DB) (err error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return err
	}

	base.ID = id

	return nil
}

type Base struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key;"`
	CreatedAt time.Time  `json:"-"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `sql:"index" json:"-"`
}
