package main

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func ProductIsOrdered(productID string) bool {
	err := db.Where("product_id = ?", productID).Take(&OrderedProduct{}).Error
	return err == nil
}

func CreateProductCopy(product Product) Product {
	var productCopy Product
	if err := db.Where(Product{BaseProductID: &product.ID}).Take(&productCopy).Error; err != nil {
		productCopy = product

		productCopy.BaseProductID = new(string)
		*productCopy.BaseProductID = product.ID
		productCopy.ID = ""
		productCopy.Public = false
		db.Create(&productCopy)
	}

	return productCopy
}

func GetPublicOrOwnerProducts(tx *gorm.DB, r *http.Request) {
	email := GetClaim("email", r)

	statement := "public = ?"
	shopID := ""
	if email != nil {
		var shop Shop

		err := GetShopByEmail(*email, &shop, false, "id")
		if err == nil {
			statement += " OR shop_id = ?"
			shopID = shop.ID
		}
	}

	tx.Where("base_product_id is null").Where(statement, true, shopID)
}

func GetProducts(w http.ResponseWriter, r *http.Request) {
	products := make([]Product, 0)

	tx := db.Preload(clause.Associations)
	GetPublicOrOwnerProducts(tx, r)

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
		db.Table("product_categories").Select("product_id").Where("category_id IN ?", categoryIDs).Find(&productIDs)

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

	tx.Order("created_at desc").Find(&products)
	JSONResponse(products, w)
}

func GetProduct(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	productName := params["product"]

	var product Product

	tx := db.Preload(clause.Associations)
	GetPublicOrOwnerProducts(tx, r)

	err := tx.Where("codename = ?", productName).Take(&product).Error
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	JSONResponse(product, w)
}

func AddEditProduct(w http.ResponseWriter, r *http.Request) {
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
			Response(w, http.StatusBadRequest, "prieš kuriant prekę privalote susikurti parduotuvę")
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
		Response(w, http.StatusBadRequest, "blogi duomenys, bandykite dar kartą")
		return
	}

	var productCopy Product
	if isEdit {
		db.Preload("Categories").Where("codename = ?", productName).Take(&product)
		productCopy = product
	}

	// Name
	if !isEdit && (request.Name == nil || len(*request.Name) == 0) {
		Response(w, http.StatusBadRequest, "produkto vardas privalomas")
		return
	}

	if request.Name != nil {
		product.Name = request.Name
		product.Codename = GenerateCodename(*request.Name, true)
	}

	// Quantity
	if request.Quantity != nil && *request.Quantity < 0 {
		Response(w, http.StatusBadRequest, "kiekis turi būti didesnis už 0")
		return
	}

	if !isEdit && request.Quantity == nil {
		Response(w, http.StatusBadRequest, "kiekis yra privalomas")
		return
	}

	if request.Quantity != nil {
		product.Quantity = *request.Quantity
	}

	// Amount
	if request.Price != nil && *request.Price < -1 {
		Response(w, http.StatusBadRequest, "kaina turi būti didesnis už 0")
		return
	}

	if !isEdit && request.Price == nil {
		Response(w, http.StatusBadRequest, "kaina yra privaloma")
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

	if !isEdit && product.Description == nil {
		product.Description = new(string)
	}

	if isEdit {
		OnProductChange(productCopy)
	}

	product.ID = ""

	if err = db.Create(&product).Error; err != nil {
		Response(w, http.StatusBadRequest, "įvyko klaida bandant išsaugoti produktą")
		return
	}

	if isEdit {
		db.Table("product_categories").Where("product_id = ?", productCopy.ID).Update("product_id", product.ID)
	} else {
		w.WriteHeader(http.StatusCreated)
	}

	JSONResponse(product, w)
}

func DeleteProduct(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	productName := params["product"]

	var product Product
	err := db.Take(&product, "codename = ?", productName).Error

	if err != nil {
		Response(w, http.StatusBadRequest, "įvyko klaida trintant produktą")
		return
	}

	OnProductChange(product)
}
