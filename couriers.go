package main

import (
	"net/http"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func GetDeliveries(w http.ResponseWriter, r *http.Request) {
	email := GetClaim("email", r)
	var courier User

	if err := db.Take(&courier, "email = ?", email).Error; err != nil {
		Response(w, http.StatusBadRequest, "toks kurjeris neegzistuoja")
		return
	}

	var orders []Order

	tx := db.Unscoped().Preload(clause.Associations)
	tx.Preload("ShopOrders", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at desc")
	})

	tx.Preload("ShopOrders.OrderedProducts").Preload("ShopOrders.Collector")
	tx.Preload("ShopOrders.OrderedProducts.Product").Preload("ShopOrders.Shop")
	tx.Where("status > ?", 2).Where("status < ?", 5).Where("delivered_by = ?", courier.ID)

	tx.Order("created_at desc").Find(&orders)
	JSONResponse(orders, w)
}

func GetPickups(w http.ResponseWriter, r *http.Request) {
	email := GetClaim("email", r)
	var courier User

	if err := db.Take(&courier, "email = ?", email).Error; err != nil {
		Response(w, http.StatusBadRequest, "toks kurjeris neegzistuoja")
		return
	}

	var shopOrders []ShopOrder

	tx := db.Unscoped().Preload(clause.Associations).Preload("OrderedProducts").Preload("OrderedProducts.Product")

	tx.Joins("left join orders on orders.id = shop_orders.order_id").Where("orders.pickup_date > ?", time.Now())
	tx.Where("shop_orders.status < ?", 3).Where("collected_by = ?", courier.ID).Order("orders.pickup_date").Find(&shopOrders)

	JSONResponse(shopOrders, w)
}
