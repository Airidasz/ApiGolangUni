package main

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type ctxKey struct{}

type User struct {
	ID           uuid.UUID `gorm:"primary_key;type:uuid;default:uuid_generate_v4()"`
	Name         string    `json:"name" gorm:"size:100"`
	Email        string    `json:"email" gorm:"size:100;not null"`
	Password     string    `json:"-" gorm:"size:100;not null"`
	Salt         string    `json:"-" gorm:"size:64;not null"`
	Permissions  string    `json:"-" gorm:"size:20"`
	ShopCodename *string   `json:"-"`
	Temporary    bool      `json:"temporary"`
}

type Shop struct {
	ID          uuid.UUID  `json:"-" gorm:"primary_key;type:uuid;default:uuid_generate_v4()"`
	Name        *string    `json:"name" gorm:"size:100;not null"`
	Codename    string     `json:"codename" gorm:"size:100;not null"`
	Description *string    `json:"description"`
	User        User       `json:"-" gorm:"not null"`
	UserID      uuid.UUID  `json:"-"`
	Locations   []Location `json:"-" gorm:"constraint:OnDelete:CASCADE;"`
	Products    []Product  `json:"-" gorm:"constraint:OnDelete:CASCADE;"`
}

type Product struct {
	ID          uuid.UUID       `json:"-" gorm:"primary_key;type:uuid;default:uuid_generate_v4()"`
	Name        *string         `json:"name" gorm:"size:100;not null"`
	Codename    string          `json:"codename" gorm:"size:100;not null"`
	Description *string         `json:"description"`
	Image       string          `json:"image" gorm:"size:500"`
	Amount      decimal.Decimal `json:"amount" sql:"type:decimal(20,8);"  gorm:"not null"`
	Public      bool            `json:"public"`
	Quantity    int             `json:"quantity" gorm:"not null"`
	Shop        Shop            `json:"shop" gorm:"not null"`
	ShopID      uuid.UUID       `json:"-" gorm:"not null"`
	Categories  []Category      `json:"categories" gorm:"many2many:product_categories;constraint:OnDelete:CASCADE;"`
}

type Location struct {
	ID     uuid.UUID `gorm:"primary_key;type:uuid;default:uuid_generate_v4()"`
	Type   string    `json:"type"  gorm:"size:30;not null"`
	Lat    float32   `json:"lat"  gorm:"not null"`
	Lng    float32   `json:"lng"  gorm:"not null"`
	Shop   Shop      `json:"-" gorm:"not null"`
	ShopID uuid.UUID `json:"-"`
}

type Order struct {
	ID       uuid.UUID `gorm:"primary_key;type:uuid;default:uuid_generate_v4()"`
	Email    string    `json:"email" gorm:"size:100;not null"`
	Status   int       `json:"status" gorm:"not null"`
	Note     string    `json:"note" gorm:"size:100;not null"`
	Shipping string    `json:"shipping"`
	Payment  string    `json:"payment" gorm:"size:100"`
	// OrderedProducts []OrderedProduct `json:"orderedProducts" gorm:"foreignKey:ID;"`
}

type OrderedProduct struct {
	ID        uuid.UUID `gorm:"primary_key;type:uuid;default:uuid_generate_v4()"`
	Order     Order     `json:"order" gorm:"not null"`
	OrderID   uuid.UUID `json:"-" gorm:"not null"`
	Product   Product   `json:"product" gorm:"not null"`
	ProductID uuid.UUID `json:"-" gorm:"not null"`
	Quantity  int       `json:"quantity" gorm:"not null"`
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
