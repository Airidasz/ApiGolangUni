package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/shopspring/decimal"
	"gorm.io/gorm/clause"
)

func GetOrders(w http.ResponseWriter, r *http.Request) {
	email := GetClaim("email", r)
	permissions := strings.ToLower(*GetClaim("permissions", r))

	var orders []Order

	tx := db.Unscoped().Preload(clause.Associations).Preload("OrderedProducts").Preload("OrderedProducts.Product")

	// Only filter if user is not an admin
	if !strings.Contains(permissions, "a") {
		tx.Where("email = ?", email)
	}

	tx.Order("created_at desc").Find(&orders)
	JSONResponse(orders, w)
}

func PlaceOrder(w http.ResponseWriter, r *http.Request) {
	var errorStruct ErrorJSON

	// Decode request
	request := struct {
		User            User             `json:"user"`
		Note            string           `json:"note"`
		ShippingType    *int             `json:"shippingType"`
		Address         *string          `json:"address"`
		PaymentType     *int             `json:"paymentType"`
		OrderedProducts []OrderedProduct `json:"orderedProducts"`
	}{User{}, "", nil, nil, nil, nil}

	err := json.NewDecoder(r.Body).Decode(&request)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "unable to parse body to json"
		JSONResponse(errorStruct, w)
		return
	}

	if request.ShippingType == nil || request.Address == nil {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "shipping information missing"
		JSONResponse(errorStruct, w)
		return
	}

	if request.PaymentType == nil {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "payment information missing"
		JSONResponse(errorStruct, w)
		return
	}

	if len(request.User.Email) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "email missing"
		JSONResponse(errorStruct, w)
		return
	}

	// Check for errors:
	// does product exist, is the quantity correct
	var orderedProducts []OrderedProduct
	errors := make(map[string]string)

	totalPrice := decimal.Zero
	productCache := make(map[string]Product)
	for _, orderedProduct := range request.OrderedProducts {
		var product Product

		err := db.Take(&product, "codename = ?", orderedProduct.Product.Codename).Error
		if err != nil {
			errors[orderedProduct.Product.Codename] = "Product doesn't exist"
			continue
		}

		if orderedProduct.Quantity > product.Quantity {
			errors[orderedProduct.Product.Codename] = fmt.Sprintf("Product only has %d available units", product.Quantity)
			continue
		}

		newProductOrder := OrderedProduct{
			UnitPrice: product.Price,
			ProductID: product.ID,
			Quantity:  orderedProduct.Quantity,
		}

		quantityDecimal := decimal.NewFromInt(int64(orderedProduct.Quantity))
		totalPrice = totalPrice.Add(product.Price.Mul(quantityDecimal))

		orderedProducts = append(orderedProducts, newProductOrder)

		productCache[product.ID] = product
	}

	// Return errors
	if len(errors) > 0 {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "order invalid"
		errorStruct.Payload = errors
		JSONResponse(errorStruct, w)
		return
	}

	if request.User.Temporary {
		// create temp user
		err = CreateTempUser(request.User)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			errorStruct.Message = err.Error()
			JSONResponse(errorStruct, w)
			return
		}
	}

	// No errors, create order
	order := Order{
		Codename:     GenerateOrderIdentifier(),
		Email:        request.User.Email,
		Status:       1,
		Note:         request.Note,
		ShippingType: *request.ShippingType,
		Address:      *request.Address,
		PaymentType:  *request.PaymentType,
		TotalPrice:   totalPrice.Round(2),
	}

	db.Create(&order)

	for _, orderedProduct := range orderedProducts {
		product := productCache[orderedProduct.ProductID]
		product.Quantity -= orderedProduct.Quantity
		db.Save(&product)

		orderedProduct.OrderID = order.ID
		db.Create(&orderedProduct)
	}

	w.WriteHeader(http.StatusCreated)
}

func ChangeOrder(w http.ResponseWriter, r *http.Request) {
	var errorStruct ErrorJSON

	request := struct {
		Status   *int    `json:"status"`
		Codename *string `json:"codename"`
	}{nil, nil}

	err := json.NewDecoder(r.Body).Decode(&request)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "unable to parse body to json"
		JSONResponse(errorStruct, w)
		return
	}

	if request.Status == nil || request.Codename == nil {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "status or codename not set"
		JSONResponse(errorStruct, w)
		return
	}

	var order Order

	db.Take(&order, "codename = ?", *request.Codename)
	order.Status = *request.Status
	db.Save(&order)
}
