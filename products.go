package main

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/shopspring/decimal"
	"gorm.io/gorm/clause"
)

func GetProducts(w http.ResponseWriter, r *http.Request) {
	products := make([]Product, 0)

	tx := db.Preload(clause.Associations)

	r.ParseForm()

	requestedCategories := r.Form["category"]

	if len(requestedCategories) > 0 {
		var categoryIDs []string
		db.Model(&Category{}).Select("id").Where("codename IN ?", requestedCategories).Find(&categoryIDs)
		if len(categoryIDs) == 0 {
			JSONResponse(products, w)
			return
		}

		var productIDs []string
		db.Table("product_categories").Select("product_id").Where("category_id IN ?", categoryIDs).Take(&productIDs)

		if len(productIDs) == 0 {
			JSONResponse(products, w)
			return
		}

		tx.Where("id in ?", productIDs)
	}

	requestedShops := r.Form["shop"]

	if len(requestedShops) > 0 {
		var shopIDs []string
		db.Model(&Shop{}).Select("id").Where("codename IN ?", requestedShops).Find(&shopIDs)

		if len(shopIDs) == 0 {
			JSONResponse(products, w)
			return
		}

		tx.Where("shop_id in ?", shopIDs)
	}

	tx.Where(Product{Public: true})

	email := GetClaim("email", r)

	if email != nil {
		var shop Shop

		err := GetShopByEmail(*email, &shop, false, "id")
		if err == nil {
			tx.Or("shop_id = ?", shop.ID)
		}
	}

	tx.Order("created_at desc").Find(&products)
	JSONResponse(products, w)
}

func GetProduct(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	productName := params["product"]

	var product Product

	err := db.Preload(clause.Associations).Where("codename = ?", productName).Take(&product).Error
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	JSONResponse(product, w)
}

func AddEditProduct(w http.ResponseWriter, r *http.Request) {
	var errorStruct ErrorJSON

	params := mux.Vars(r)
	productName := params["product"]
	var product Product

	isEdit := len(productName) > 0

	if !isEdit {
		email := GetClaim("email", r)

		var user User
		db.Take(&user, "email = ?", email)

		var shop Shop
		db.Where("user_id = ?", user.ID).Take(&shop)

		if shop.Name == nil {
			w.WriteHeader(http.StatusBadRequest)
			errorStruct.Message = "no shop detected, please create a shop before creating a product"
			JSONResponse(errorStruct, w)
			return
		}

		product.ShopID = shop.ID
	}

	r.ParseMultipartForm(10 << 20)
	image := FileUpload(r, "file", "product-*.png")

	request := struct {
		Name        *string   `json:"name"`
		Description *string   `json:"description"`
		Categories  *[]string `json:"categories"`
		Price       *float64  `json:"amount"`
		Public      *bool     `json:"public"`
		Quantity    *int      `json:"quantity"`
	}{nil, nil, nil, nil, nil, nil}

	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	decoder.ZeroEmpty(true)
	decoder.RegisterConverter([]string{}, func(input string) reflect.Value {
		return reflect.ValueOf(strings.Split(input, ","))
	})

	err := decoder.Decode(&request, r.Form)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = err.Error()
		JSONResponse(errorStruct, w)
		return
	}

	if isEdit {
		db.Preload("Categories").Where("codename = ?", productName).Take(&product)
	}

	// Name
	if !isEdit && (request.Name == nil || len(*request.Name) == 0) {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "product name cannot be empty"
		JSONResponse(errorStruct, w)
		return
	}

	if request.Name != nil {
		product.Name = request.Name
		product.Codename = GenerateCodename(*request.Name, true)
	}

	// Quantity
	if request.Quantity != nil && *request.Quantity < 0 {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "quantity cannot be less than zero"
		JSONResponse(errorStruct, w)
		return
	}

	if !isEdit && request.Quantity == nil {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "you must set a quantity"
		JSONResponse(errorStruct, w)
		return
	}

	if request.Quantity != nil {
		product.Quantity = *request.Quantity
	}

	// Amount
	if request.Price != nil && *request.Price < -1 {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "price cannot be less than zero"
		JSONResponse(errorStruct, w)
		return
	}

	if !isEdit && request.Price == nil {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "you must set a price"
		JSONResponse(errorStruct, w)
		return
	}

	if request.Price != nil {
		product.Price = decimal.NewFromFloat(*request.Price)
	}

	// Description
	if request.Description != nil {
		product.Description = request.Description
	}

	// Public product
	if request.Public != nil {
		product.Public = *request.Public
	}

	// Categories
	if request.Categories != nil {
		if isEdit {
			db.Exec("DELETE FROM product_categories WHERE product_id = ?", product.ID)
		}

		var categories []Category
		db.Find(&categories, "id in ?", *request.Categories)
		product.Categories = categories
	}

	// Image
	if len(image) > 0 {
		product.Image = image
	}

	// Add or save to Database
	if isEdit {
		err = db.Save(&product).Error
	} else {
		err = db.Create(&product).Error
	}

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "you must set a price"
		JSONResponse(errorStruct, w)
		return
	}

	JSONResponse(product, w)
}

func DeleteProduct(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	productName := params["product"]

	db.Unscoped().Where("codename = ?", productName).Delete(&Product{})
}
