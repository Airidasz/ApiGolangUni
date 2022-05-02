package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func OnProductChange(product Product) {
	if ProductIsOrdered(product.ID) {
		product.BaseProductID = new(string)
		*product.BaseProductID = product.ID
		product.Public = false
		db.Save(&product)
		db.Delete(&product)
	} else {
		db.Unscoped().Delete(&product)
	}
}

func GetOrders(w http.ResponseWriter, r *http.Request) {
	email := GetClaim("email", r)
	permissions := strings.ToLower(*GetClaim("permissions", r))

	var orders []Order

	tx := db.Unscoped().Preload(clause.Associations)
	tx.Preload("ShopOrders", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at desc")
	})

	tx.Preload("ShopOrders.OrderedProducts").Preload("ShopOrders.Collector")
	tx.Preload("ShopOrders.OrderedProducts.Product").Preload("ShopOrders.Shop")
	// Only filter if user is not an admin
	if !strings.ContainsAny(permissions, "aA") {
		tx.Where("email = ?", email)
	}

	tx.Order("created_at desc").Find(&orders)
	JSONResponse(orders, w)
}

func PlaceOrder(w http.ResponseWriter, r *http.Request) {
	// Decode request
	request := struct {
		User            User             `json:"user"`
		Note            string           `json:"note"`
		Address         *string          `json:"address"`
		PaymentType     *int             `json:"paymentType"`
		CancelIfMissing bool             `json:"cancelIfMissing"`
		OrderedProducts []OrderedProduct `json:"orderedProducts"`
	}{User{}, "", nil, nil, false, nil}

	err := json.NewDecoder(r.Body).Decode(&request)

	if err != nil {
		Response(w, http.StatusBadRequest,"blogas duomenų formatas")
		return
	}

	if request.Address == nil {
		Response(w, http.StatusBadRequest,"adresas yra privalomas")
		return
	}

	if request.PaymentType == nil {
		Response(w, http.StatusBadRequest,"mokėjimo informacija yra privaloma")
		return
	}

	if len(request.User.Email) == 0 {
		Response(w, http.StatusBadRequest,"el.pašto adresas yra privalomas")
		return
	}

	// Check for errors:
	// does product exist, is the quantity correct
	var orderedProducts []OrderedProduct
	errors := make(map[string]string)

	totalPrice := decimal.Zero
	productCache := make(map[string]Product)
	shopsWithOrders := make(map[string]bool)

	for _, orderedProduct := range request.OrderedProducts {
		var product Product

		err := db.Take(&product, "codename = ?", orderedProduct.Product.Codename).Error
		if err != nil {
			errors[orderedProduct.Product.Codename] = "produkto nepavyko rasti"
			continue
		}

		if orderedProduct.Quantity > product.Quantity {
			errors[orderedProduct.Product.Codename] = fmt.Sprintf("produktas turi tik %d likusius vientos", product.Quantity)
			continue
		}

		newProductOrder := OrderedProduct{
			ProductID: product.ID,
			Quantity:  orderedProduct.Quantity,
		}

		quantityDecimal := decimal.NewFromInt(int64(orderedProduct.Quantity))
		totalPrice = totalPrice.Add(product.Price.Mul(quantityDecimal))

		orderedProducts = append(orderedProducts, newProductOrder)

		shopsWithOrders[product.ShopID] = true
		productCache[product.ID] = product
	}

	// Return errors
	if len(errors) > 0 {
		Response(w, http.StatusBadRequest,"įvyko klaida sukuriant užsakymą", errors)
		return
	}

	if request.User.Temporary {
		// create temp user
		err = CreateTempUser(request.User)

		if err != nil {
			Response(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	// No errors, create order
	order := Order{
		Codename:        GenerateOrderIdentifier(),
		Email:           request.User.Email,
		Status:          1,
		Note:            request.Note,
		Address:         *request.Address,
		PaymentType:     *request.PaymentType,
		TotalPrice:      totalPrice.Round(2),
		CancelIfMissing: request.CancelIfMissing,
	}

	db.Create(&order)

	shopOrders := make(map[string]string)
	// Create ordered products
	for shopID, _ := range shopsWithOrders {
		shopOrder := ShopOrder{
			OrderID: order.ID,
			ShopID:  shopID,
		}
		db.Create(&shopOrder)

		shopOrders[shopID] = shopOrder.ID
	}

	// Create ordered products
	for _, orderedProduct := range orderedProducts {
		// Reduce quantity
		product := productCache[orderedProduct.ProductID]
		product.Quantity -= orderedProduct.Quantity
		db.Save(&product)

		//productCopy := CreateProductCopy(product)

		// Set parameters for order
		orderedProduct.OrderID = order.ID
		orderedProduct.ProductID = product.ID
		orderedProduct.ShopOrderID = shopOrders[product.ShopID]
		db.Create(&orderedProduct)
	}

	w.WriteHeader(http.StatusCreated)
	JSONResponse(order, w)
}

func GetCouriers(w http.ResponseWriter, r *http.Request) {
	var couriers []User
	db.Where("permissions ~ ?", "c").Find(&couriers)
	JSONResponse(couriers, w)
}

func CancelOrder(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	orderNumber := params["ordernumber"]

	request := struct {
		Status     *int    `json:"status"`
		Deliverer  *string `json:"deliverer"`
		PickupDate *string `json:"pickupDate"`
	}{nil, nil, nil}

	err := json.NewDecoder(r.Body).Decode(&request)

	if err != nil {
		Response(w, http.StatusBadRequest, "blogas duomenų formatas")
		return
	}

	var order Order
	email := GetClaim("email", r)
	err = db.Take(&order, "codename = ? AND email = ?", orderNumber, email).Error
	if err != nil {
		Response(w, http.StatusBadRequest, "užsakymas nerastas")
		return
	}

	order.Status = 5
	err = db.Save(&order).Error
	if err != nil {
		Response(w, http.StatusInternalServerError, "klaida saugojant duomenis. bandykite dar kartą")
		return
	}
}

func ChangeOrder(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	orderNumber := params["ordernumber"]

	request := struct {
		Status     *int    `json:"status"`
		Deliverer  *string `json:"deliverer"`
		PickupDate *string `json:"pickupDate"`
	}{nil, nil, nil}

	err := json.NewDecoder(r.Body).Decode(&request)

	if err != nil {
		Response(w, http.StatusBadRequest, "blogas duomenų formatas")
		return
	}

	var order Order

	err = db.Take(&order, "codename = ?", orderNumber).Error
	if err != nil {
		Response(w, http.StatusBadRequest, "užsakymas nerastas")
		return
	}

	if request.Status != nil {
		order.Status = *request.Status
	}

	if request.Deliverer != nil {
		var delivererUser User
		err = db.Take(&delivererUser, "email = ?", request.Deliverer).Error

		if err == nil {
			order.DeliveredBy = delivererUser.ID
		}
	}

	if request.PickupDate != nil {
		parsedDate, dateErr := time.Parse("2006-01-02", *request.PickupDate)

		if dateErr == nil {
			order.PickupDate = &parsedDate
		}
	}

	err = db.Save(&order).Error

	if err != nil {
		Response(w, http.StatusInternalServerError, "klaida saugojant duomenis. bandykite dar kartą")
		return
	}

	OnOrderChange(order)
}

func DeleteTempUser(email string) {
	var user User
	err := db.Where("email = ? AND temporary = ?", email, true).Take(&user).Error
	if err == nil {
		db.Unscoped().Delete(&user)
	}
}

func OnShopOrderChange(shopOrder ShopOrder) {
	if shopOrder.Status == 1 {
		var shopOrders []ShopOrder
		if db.Where("status = 0").Find(&shopOrders).RowsAffected > 0 {
			return
		}

		db.Model(&Order{}).Where("id = ?", shopOrder.OrderID).Update("status", 3)
	} else if shopOrder.Status > 2 {
		var order Order
		db.Take(&order, "id = ?", shopOrder.OrderID)

		if order.CancelIfMissing {
			order.Status = 5
			OnOrderChange(order)
		} else {
			// Remove cancelled shop order product price from total price
			var products []map[string]interface{}
			tx := db.Unscoped().Table("products").Select("products.price", "ordered_products.quantity")
			tx.Joins("left join ordered_products on ordered_products.product_id = products.id")
			tx.Where("ordered_products.shop_order_id = ?", shopOrder.ID)
			tx.Find(&products)

			sum := decimal.Zero

			for _, product := range products {
				price, _ := decimal.NewFromString(fmt.Sprint(product["price"]))
				quantity, _ := decimal.NewFromString(fmt.Sprint(product["quantity"]))

				sum = sum.Add(price.Mul(quantity))
			}

			order.TotalPrice = order.TotalPrice.Sub(sum)
		}

		db.Save(&order)
	}
}

func OnOrderChange(order Order) {
	if order.Status == 4 {
		DeleteTempUser(order.Email)
	} else if order.Status > 4 { // Cancelled or error
		db.Model(&ShopOrder{}).Where("order_id = ?", order.ID).Update("status", 3)
	}
}
