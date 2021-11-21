package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
)

func LandingPage(w http.ResponseWriter, r *http.Request) {
	b := struct {
		Make    string `json:"make"`
		Model   string `json:"model"`
		Mileage int    `json:"mileage"`
	}{"Vienas", "Du", 1}

	JSONResponse(b, w)
}

// =========================== Auth ===================================
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	//Creates a struct used to store data decoded from the body
	requestData := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{"", ""}

	json.NewDecoder(r.Body).Decode(&requestData)

	var userDatabaseData User

	// Finds user by email in database, if no user, then returns "bad request"
	err := db.Take(&userDatabaseData, "email = ?", requestData.Email).Error
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(ErrorJSON{
			Message: "bad login information",
		}, w)
		return
	}

	hashedPassword := GenerateSecurePassword(requestData.Password, userDatabaseData.Salt)
	//checks if salted hashed password from database matches the sent in salted hashed password
	if hashedPassword != userDatabaseData.Password {
		w.WriteHeader(http.StatusUnauthorized)
		JSONResponse(ErrorJSON{
			Message: "bad login information",
		}, w)
		return
	}

	MakeTokens(w, userDatabaseData)

	w.WriteHeader(http.StatusAccepted)
	JSONResponse(userDatabaseData, w)
}

//CreateAccountHandler decodes user sent in data, verifies that
//it is formatted correctly, and tries to create an account in
//the database
func CreateAccountHandler(w http.ResponseWriter, r *http.Request) {
	//Creates a struct used to store data decoded from the body
	requestData := struct {
		Email          string `json:"email"`
		Password       string `json:"password"`
		RepeatPassword string `json:"repeatPassword"`
	}{"", "", ""}

	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(ErrorJSON{
			Message: err.Error(),
		}, w)
		return
	}

	res, err := PerformUserDataChecks(requestData.Email, requestData.Password, requestData.RepeatPassword)

	w.WriteHeader(res)
	if err != nil {
		JSONResponse(ErrorJSON{
			Message: err.Error(),
		}, w)
		return
	}

	salt := GenerateSalt()
	hashedPassword := GenerateSecurePassword(requestData.Password, salt)

	newUser := User{
		Email:    requestData.Email,
		Password: hashedPassword,
		Salt:     salt,
	}
	db.Save(&newUser)
	w.WriteHeader(http.StatusCreated)
}

func RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	refreshTokenCookie, err := r.Cookie("Refresh-Token")

	if err == nil {
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(refreshTokenCookie.Value, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("there was an error")
			}
			return signKey, nil
		})

		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			JSONResponse(ErrorJSON{
				Message: err.Error(),
			}, w)
			return
		}

		if !token.Valid {
			w.WriteHeader(http.StatusUnauthorized)
			JSONResponse(ErrorJSON{
				Message: err.Error(),
			}, w)
			return
		}

		email := fmt.Sprintf("%v", claims["email"])

		var oldRefreshToken RefreshToken
		db.Take(&oldRefreshToken, "token = ?", refreshTokenCookie.Value)

		if oldRefreshToken.DeletedAt.Valid {
			db.Delete(&RefreshToken{}, "email = ?", email)
			w.WriteHeader(http.StatusForbidden)
			JSONResponse(ErrorJSON{
				Message: "expired token",
			}, w)
			return
		}

		if exp, ok := claims["exp"].(int64); ok && exp > time.Now().Unix() {
			w.WriteHeader(http.StatusUnauthorized)
			JSONResponse(ErrorJSON{
				Message: "expired token",
			}, w)
			return
		}

		db.Delete(&oldRefreshToken)

		var user User
		db.Take(&user, "email = ?", email)

		MakeTokens(w, user)
		w.WriteHeader(http.StatusAccepted)
		return
	}

	w.WriteHeader(http.StatusUnauthorized)
	JSONResponse(ErrorJSON{
		Message: "unauthorized",
	}, w)
}

// ===================================================================

// =========================== Shop ===================================
func GetShopsHandler(w http.ResponseWriter, r *http.Request) {
	keys := r.URL.Query()
	userID, err := strconv.Atoi(keys.Get("userid"))

	tx := db.Model(&Shop{})
	if err == nil && userID > 0 {
		tx.Where("user_id = ?", userID)
	}

	var shops []Shop
	tx.Find(&shops)
	JSONResponse(shops, w)
}

func GetShopHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	shopID, err := strconv.Atoi(params["shopid"])

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(ErrorJSON{
			Message: "bad shop id",
		}, w)
		return
	}

	var shop Shop
	if db.Take(&shop, shopID).Error != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	JSONResponse(&shop, w)
}

func CreateShopHandler(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(ctxKey{}).(jwt.MapClaims)
	email := fmt.Sprintf("%v", claims["email"])

	var requestInfo Shop
	err := json.NewDecoder(r.Body).Decode(&requestInfo)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(ErrorJSON{
			Message: "json parse error",
		}, w)
		return
	}

	var user User
	db.Take(&user, "email = ?", email)

	requestInfo.User = user
	db.Create(&requestInfo)

	w.WriteHeader(http.StatusCreated)
}

func UpdateShopHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	shopID, _ := strconv.Atoi(params["shopid"])

	var requestInfo Shop
	err := json.NewDecoder(r.Body).Decode(&requestInfo)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(ErrorJSON{
			Message: "json parse error",
		}, w)
		return
	}

	tx := db.Model(&Shop{}).Where("id = ?", shopID)

	if requestInfo.Name != "" {
		tx.Updates(Shop{Name: requestInfo.Name})
	}

	if requestInfo.Description != "" {
		tx.Updates(Shop{Description: requestInfo.Description})
	}

	w.WriteHeader(http.StatusOK)
}

func DeleteShopHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	shopID, _ := strconv.Atoi(params["shopid"])

	db.Unscoped().Select("Locations", "Products").Delete(&Shop{}, shopID)
}

// ===================================================================

// ========================== Locations ==============================
func GetLocationsHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	shopID, err := strconv.Atoi(params["shopid"])

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(ErrorJSON{
			Message: "bad shop id",
		}, w)
		return
	}

	var locations []Location
	db.Where("shop_id = ?", shopID).Find(&locations)
	JSONResponse(locations, w)
}

func GetLocationHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	locationID, _ := strconv.Atoi(params["locationid"])

	var location Location

	if db.Take(&location, locationID).Error != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	JSONResponse(&location, w)
}

func CreateLocationHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	shopID, _ := strconv.Atoi(params["shopid"])

	var requestInfo Location
	err := json.NewDecoder(r.Body).Decode(&requestInfo)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(ErrorJSON{
			Message: "json parse error",
		}, w)
		return
	}

	requestInfo.ShopID = uint(shopID)
	db.Create(&requestInfo)

	w.WriteHeader(http.StatusCreated)
}

func UpdateLocationHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	locationID, _ := strconv.Atoi(params["locationid"])

	var requestInfo Location
	err := json.NewDecoder(r.Body).Decode(&requestInfo)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(ErrorJSON{
			Message: "json parse error",
		}, w)
		return
	}

	tx := db.Model(&Location{}).Where("id = ?", locationID)

	if requestInfo.Coordinates != "" {
		tx.Updates(Location{Coordinates: requestInfo.Coordinates})
	}

	w.WriteHeader(http.StatusOK)
}

func DeleteLocationHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	locationID, _ := strconv.Atoi(params["locationid"])

	db.Delete(&Location{}, locationID)
}

// ===================================================================

// ========================== Products ==============================
func GetProductsHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	shopID, err := strconv.Atoi(params["shopid"])

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(ErrorJSON{
			Message: "bad shop id",
		}, w)
		return
	}

	var products []Product
	db.Preload("Categories").Where("shop_id = ?", shopID).Find(&products)
	JSONResponse(products, w)
}

func GetProductHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	productID, _ := strconv.Atoi(params["productid"])

	var product Product

	if db.Take(&product, productID).Error != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	JSONResponse(&product, w)
}

func CreateProductHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	shopID, _ := strconv.Atoi(params["shopid"])

	requestInfo := struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Categories  []int  `json:"categories"`
	}{"", "", nil}

	err := json.NewDecoder(r.Body).Decode(&requestInfo)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(ErrorJSON{
			Message: "json parse error",
		}, w)
		return
	}

	var categories []Category
	db.Find(&categories, requestInfo.Categories)

	newShop := Product{
		Name:        requestInfo.Name,
		Description: requestInfo.Description,
		ShopID:      uint(shopID),
		Categories:  categories,
	}
	db.Create(&newShop)

	w.WriteHeader(http.StatusCreated)
}

func UpdateProductHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	productID, _ := strconv.Atoi(params["productid"])

	requestInfo := struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Categories  []int  `json:"categories"`
	}{"", "", nil}

	err := json.NewDecoder(r.Body).Decode(&requestInfo)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(ErrorJSON{
			Message: "json parse error",
		}, w)
		return
	}

	var product Product
	db.Preload("Categories").Take(&product, productID)

	if requestInfo.Name != "" {
		db.Model(&product).Updates(Product{Name: requestInfo.Name})
	}

	if requestInfo.Description != "" {
		db.Model(&product).Updates(Product{Description: requestInfo.Description})
	}

	if len(requestInfo.Categories) > 0 {
		var categories []Category
		db.Find(&categories, requestInfo.Categories)

		db.Model(&product).Association("Categories").Delete(&product.Categories)
		db.Model(&product).Association("Categories").Append(&categories)

		db.Model(&product).Updates(Product{Categories: categories})
	}

	w.WriteHeader(http.StatusOK)
}

func DeleteProductHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	productID, _ := strconv.Atoi(params["productid"])

	var product Product
	db.Preload("Categories").Take(&product, productID)

	db.Model(&product).Association("Categories").Delete(&product.Categories)
	db.Delete(&product)
}

// ===================================================================

// ========================== Categories ==============================
func GetCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	var categories []Category
	db.Find(&categories)
	JSONResponse(categories, w)
}

func GetCategoryHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	caregoryID, err := strconv.Atoi(params["categoryid"])

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(ErrorJSON{
			Message: "bad category id",
		}, w)
		return
	}

	var category Category

	if db.Take(&category, caregoryID).Error != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	JSONResponse(&category, w)
}

func CreateCategoryHandler(w http.ResponseWriter, r *http.Request) {
	var requestInfo Category
	err := json.NewDecoder(r.Body).Decode(&requestInfo)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(ErrorJSON{
			Message: "json parse error",
		}, w)
		return
	}

	db.Create(&requestInfo)

	w.WriteHeader(http.StatusCreated)
}

func UpdateCategoryHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	categoryID, err := strconv.Atoi(params["categoryid"])

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(ErrorJSON{
			Message: "bad category id",
		}, w)
		return
	}

	var requestInfo Category
	err = json.NewDecoder(r.Body).Decode(&requestInfo)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(ErrorJSON{
			Message: "json parse error",
		}, w)
		return
	}

	tx := db.Model(&Category{}).Where("id = ?", categoryID)

	if requestInfo.Name != "" {
		tx.Updates(Shop{Name: requestInfo.Name})
	}

	w.WriteHeader(http.StatusOK)
}

func DeleteCategoryHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	caregoryID, err := strconv.Atoi(params["categoryid"])

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(ErrorJSON{
			Message: "bad category id",
		}, w)
		return
	}

	db.Unscoped().Delete(&Category{}, caregoryID)
}
