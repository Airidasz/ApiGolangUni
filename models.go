package main

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type ctxKey struct{}

type User struct {
	ID           string    `gorm:"primary_key;size:40;default:uuid_generate_v4()"`
	CreatedAt    time.Time `json:"-"`
	Name         string    `json:"name" gorm:"size:100"`
	Email        string    `json:"email" gorm:"size:100;not null"`
	Password     string    `json:"-" gorm:"size:100;not null"`
	Salt         string    `json:"-" gorm:"size:64;not null"`
	Permissions  string    `json:"-" gorm:"size:20"`
	ShopCodename *string   `json:"-"`
	Temporary    bool      `json:"temporary"`
}

type Shop struct {
	ID          string     `json:"-" gorm:"primary_key;size:40;default:uuid_generate_v4()"`
	CreatedAt   time.Time  `json:"-"`
	Name        *string    `json:"name" gorm:"size:100;not null"`
	Codename    string     `json:"codename" gorm:"size:100;not null"`
	Description *string    `json:"description"`
	User        User       `json:"-" gorm:"not null"`
	UserID      string     `json:"-"`
	Locations   []Location `json:"locations" gorm:"constraint:OnDelete:CASCADE;"`
	Products    []Product  `json:"-" gorm:"constraint:OnDelete:CASCADE;"`
}

type Product struct {
	ID          string          `json:"-" gorm:"primary_key;size:40;default:uuid_generate_v4()"`
	CreatedAt   time.Time       `json:"-"`
	DeletedAt   gorm.DeletedAt  `json:"-" gorm:"index"`
	Name        *string         `json:"name" gorm:"size:100;not null"`
	Codename    string          `json:"codename" gorm:"size:100;not null"`
	Description *string         `json:"description"`
	Image       string          `json:"image" gorm:"size:500"`
	Price       decimal.Decimal `json:"price" sql:"type:decimal(20,8);"  gorm:"not null"`
	Public      bool            `json:"public"`
	Quantity    int             `json:"quantity" gorm:"not null"`
	Shop        Shop            `json:"shop" gorm:"not null"`
	ShopID      string          `json:"-" gorm:"not null"`
	Categories  []Category      `json:"categories" gorm:"many2many:product_categories;constraint:OnDelete:CASCADE;"`
}

type Location struct {
	ID        string    `gorm:"primary_key;size:40;default:uuid_generate_v4()"`
	CreatedAt time.Time `json:"-"`
	Type      string    `json:"type"  gorm:"size:30;not null"`
	Lat       float32   `json:"lat"  gorm:"not null"`
	Lng       float32   `json:"lng"  gorm:"not null"`
	Shop      Shop      `json:"-" gorm:"not null"`
	ShopID    string    `json:"-"`
}

type Order struct {
	ID              string           `json:"-" gorm:"primary_key;size:40;default:uuid_generate_v4()"`
	CreatedAt       time.Time        `json:"-"`
	Codename        string           `json:"codename"`
	Email           string           `json:"email" gorm:"size:100;not null"`
	Status          int              `json:"status" gorm:"not null"`
	Note            string           `json:"note" gorm:"size:100;not null"`
	ShippingType    int              `json:"shippingType"`
	Address         string           `json:"address"`
	PaymentType     int              `json:"paymentType"`
	OrderedProducts []OrderedProduct `json:"orderedProducts"`
	TotalPrice      decimal.Decimal  `json:"totalPrice"`
}

type OrderedProduct struct {
	ID        string    `json:"-" gorm:"primary_key;size:40;default:uuid_generate_v4()"`
	CreatedAt time.Time `json:"-"`
	Order     Order     `json:"order" gorm:"not null"`
	OrderID   string    `json:"-" gorm:"not null"`
	Product   Product   `json:"product" gorm:"not null"`
	ProductID string    `json:"-" gorm:"not null"`
	UnitPrice decimal.Decimal
	Quantity  int `json:"quantity" gorm:"not null"`
}

type Category struct {
	ID        string    `json:"id" gorm:"primary_key;size:40;default:uuid_generate_v4()"`
	CreatedAt time.Time `json:"-"`
	Name      *string   `json:"name" gorm:"size:100;not null"`
	Codename  string    `json:"codename" gorm:"size:100;not null"`
	File      string    `json:"file" gorm:"size:500"`
	Products  []Product `json:"-" gorm:"many2many:product_categories;constraint:OnDelete:CASCADE;"`
}

type RefreshToken struct {
	Token     string         `gorm:"primaryKey"`
	Email     string         `gorm:"not null"`
	CreatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type ErrorJSON struct {
	Message string      `json:"message"`
	Payload interface{} `json:"payload"`
}
