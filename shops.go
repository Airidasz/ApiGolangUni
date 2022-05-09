package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
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

	var shopOrders []ShopOrder

	tx := db.Preload(clause.Associations).Preload("Order", func(db *gorm.DB) *gorm.DB {
		return db.Order("pickup_date")
	}).Preload("OrderedProducts").Preload("OrderedProducts.Product", func(db *gorm.DB) *gorm.DB {
		return db.Unscoped()
	})

	tx.Where("shop_id = ?", shop.ID).Find(&shopOrders)

	JSONResponse(shopOrders, w)
}

func EditShopOrder(w http.ResponseWriter, r *http.Request) {
	var user User
	email := GetClaim("email", r)
	db.Take(&user, "email = ?", email)

	admin := HasAdminPermissions(user.Permissions)
	courier := HasCourierPermissions(user.Permissions)
	farmer := HasFarmerPermissions(user.Permissions)

	if !admin && !courier && !farmer {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var shop Shop
	if !admin && !courier {
		GetShopByEmail(*email, &shop, false, "id")
	}

	params := mux.Vars(r)
	shopOrderID := params["id"]

	var shopOrder ShopOrder
	db.Take(&shopOrder, "id = ?", shopOrderID)

	if !admin && !courier && shopOrder.ShopID != shop.ID {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	request := struct {
		Status    *int    `json:"status"`
		Message   *string `json:"message"`
		Collector *string `json:"collector"`
	}{nil, nil, nil}

	err := json.NewDecoder(r.Body).Decode(&request)

	if err != nil {
		Response(w, http.StatusBadRequest, "blogas duomenų formatas")
		return
	}

	if request.Status != nil {
		shopOrder.Status = *request.Status
	}

	if request.Message != nil {
		shopOrder.Message = *request.Message
	}

	if admin && request.Collector != nil {
		var collector User
		err = db.Take(&collector, "email = ?", request.Collector).Error

		if err == nil {
			shopOrder.CollectedBy = collector.ID
		}
	}

	err = db.Save(&shopOrder).Error
	if err != nil {
		Response(w, http.StatusInternalServerError, "klaida saugojant duomenis. bandykite dar kartą")
		return
	}

	OnShopOrderChange(shopOrder)
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
	email := GetClaim("email", r)
	err := GetShopByEmail(*email, &Shop{}, false)
	if err == nil {
		Response(w, http.StatusBadRequest, "galite turėti tik vieną parduotuvę")
		return
	}

	// Parse json to object
	var shop Shop
	err = json.NewDecoder(r.Body).Decode(&shop)

	if err != nil {
		Response(w, http.StatusBadRequest, "blogas duomenų formatas")
		return
	}

	// Info validation
	if shop.Name == nil || *shop.Name == "" {
		Response(w, http.StatusBadRequest, "vardas yra privalomas")
		return
	}

	if shop.Address == nil || *shop.Address == "" {
		Response(w, http.StatusBadRequest, "adresas yra privalomas")
		return
	}

	err = NameTaken(*shop.Name, &Shop{})
	if err != nil {
		Response(w, http.StatusConflict, err.Error())
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
		Response(w, http.StatusInternalServerError, "klaida saugojant duomenis. bandykite dar kartą")
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
	// Check if user has a shop
	email := GetClaim("email", r)

	var shop Shop
	err := GetShopByEmail(*email, &shop, false)

	if err != nil {
		Response(w, http.StatusBadRequest, "jūs neturite parduotuvės")
		return
	}

	var request Shop
	err = json.NewDecoder(r.Body).Decode(&request)

	if err != nil {
		Response(w, http.StatusBadRequest, "blogas duomenų formatas")
		return
	}

	if request.Name != nil && *request.Name == "" {
		Response(w, http.StatusBadRequest, "vardas yra privalomas")
		return
	}

	if request.Address != nil && *request.Address == "" {
		Response(w, http.StatusBadRequest, "Adresas yra privalomas")
		return
	}

	if request.Name != nil {
		err = NameTaken(*request.Name, &Shop{})
		if *shop.Name != *request.Name && err != nil {
			Response(w, http.StatusConflict, err.Error())
			return
		}

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

	if request.Address != nil {
		shop.Address = request.Address
	}

	err = db.Save(&shop).Error
	if err != nil {
		Response(w, http.StatusInternalServerError, "klaida saugojant duomenis. bandykite dar kartą")
		return
	}

	CreateLocations(shop, request.Locations)

	JSONResponse(&shop, w)
}

func CreateLocations(shop Shop, locations []Location) {
	db.Where("shop_id = ?", shop.ID).Delete(&Location{})

	for _, location := range locations {
		location.ShopID = shop.ID
		db.Create(&location)
	}
}
