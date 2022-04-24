package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"gorm.io/gorm/clause"
)

func GetShops(w http.ResponseWriter, r *http.Request) {
	var shops []Shop
	db.Order("created_at desc").Find(&shops)
	JSONResponse(shops, w)
}

func GetShopOrders(w http.ResponseWriter, r *http.Request) {
	email := GetClaim("email", r)

	var shop Shop
	GetShopByEmail(*email, &shop, false, "id")

	var orderedProducts []OrderedProduct

	db.Preload(clause.Associations).Select("ordered_products.product_id, sum(ordered_products.quantity) as quantity").Joins("left join products on products.id  = ordered_products.product_id").Where("products.shop_id = ?", shop.ID).Group("product_id").Find(&orderedProducts)
	JSONResponse(orderedProducts, w)
}

func GetShop(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	shopName := params["shop"]

	var shop Shop

	err := db.Preload(clause.Associations).Where("codename = ?", shopName).Take(&shop).Error
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	JSONResponse(&shop, w)
}

func CreateShop(w http.ResponseWriter, r *http.Request) {
	var errorStruct ErrorJSON

	email := GetClaim("email", r)
	err := GetShopByEmail(*email, &Shop{}, false)
	if err == nil {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "second shop cannot be created"
		JSONResponse(errorStruct, w)
		return
	}

	// Parse json to object
	var shop Shop
	err = json.NewDecoder(r.Body).Decode(&shop)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = err.Error()
		JSONResponse(errorStruct, w)
		return
	}

	// Info validation
	if shop.Name == nil || *shop.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "name cannot be empty"
		JSONResponse(errorStruct, w)
		return
	}

	err = NameTaken(*shop.Name, &Shop{})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = err.Error()
		JSONResponse(errorStruct, w)
		return
	}

	if shop.Description == nil {
		shop.Description = new(string)
	}

	var user User
	db.Take(&user, "email = ?", email)

	// Create shop
	shop.Codename = GenerateCodename(*shop.Name, false)
	shop.User = user
	err = db.Create(&shop).Error
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = err.Error()
		JSONResponse(errorStruct, w)
		return
	}

	// Update user info
	user.ShopCodename = &shop.Codename
	db.Save(&user)

	CreateLocations(shop, shop.Locations)

	// Send tokens with correct info
	MakeTokens(w, user)

	w.WriteHeader(http.StatusCreated)
	JSONResponse(shop, w)
}

func UpdateShop(w http.ResponseWriter, r *http.Request) {
	var errorStruct ErrorJSON

	// Check if user has a shop
	email := GetClaim("email", r)

	var shop Shop
	err := GetShopByEmail(*email, &shop, false)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "shop not found, please create a shop"
		JSONResponse(errorStruct, w)
		return
	}

	var request Shop
	err = json.NewDecoder(r.Body).Decode(&request)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = err.Error()
		JSONResponse(errorStruct, w)
		return
	}

	if request.Name != nil && *request.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "name cannot be empty"
		JSONResponse(errorStruct, w)
		return
	}

	if request.Name != nil {
		shop.Name = request.Name
		shop.Codename = GenerateCodename(*shop.Name, false)

		var user User
		db.Take(&user, "id = ?", shop.UserID)

		user.ShopCodename = &shop.Codename
		db.Save(&user)
		// Send tokens with correct info
		MakeTokens(w, user)
	}

	if request.Description != nil {
		shop.Description = request.Description
	}

	db.Save(&shop)

	CreateLocations(shop, request.Locations)

	JSONResponse(shop, w)
}

func CreateLocations(shop Shop, locations []Location) {
	db.Where("shop_id = ?", shop.ID).Delete(&Location{})

	for _, location := range locations {
		location.ShopID = shop.ID
		db.Create(&location)
	}
}

func DeleteShop(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	shopName := params["shop"]

	err := db.Select("Locations", "Products").Where("codename = ?", shopName).Delete(&Shop{}).Error
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}
