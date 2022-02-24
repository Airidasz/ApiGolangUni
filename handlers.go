package main

import (
	"encoding/json"
	"fmt"
	"net/http"
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
	var errorStruct ErrorJSON
	errorStruct.Message = "bad login information"
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
		JSONResponse(errorStruct, w)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hashedPassword := GenerateSecurePassword(requestData.Password, userDatabaseData.Salt)
	//checks if salted hashed password from database matches the sent in salted hashed password
	if hashedPassword != userDatabaseData.Password {
		JSONResponse(errorStruct, w)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	accessToken, _ := MakeTokens(w, userDatabaseData)

	w.WriteHeader(http.StatusAccepted)
	JSONResponse(struct {
		AccessToken string `json:accessToken`
	}{accessToken}, w)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "Refresh-Token", Value: "a", MaxAge: -1, SameSite: http.SameSiteNoneMode, Secure: true})
	http.SetCookie(w, &http.Cookie{Name: "Access-Token", Value: "a", MaxAge: -1, SameSite: http.SameSiteNoneMode, Secure: true})

	w.WriteHeader(http.StatusAccepted)
}

//CreateAccountHandler decodes user sent in data, verifies that
//it is formatted correctly, and tries to create an account in
//the database
func CreateAccountHandler(w http.ResponseWriter, r *http.Request) {
	var errorStruct ErrorJSON
	//Creates a struct used to store data decoded from the body
	requestData := struct {
		Name           string `json:"name"`
		Email          string `json:"email"`
		Password       string `json:"password"`
		RepeatPassword string `json:"repeatPassword"`
	}{"", "", "", ""}

	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		errorStruct.Message = err.Error()
		JSONResponse(errorStruct, w)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	res, err := PerformUserDataChecks(requestData.Email, requestData.Password, requestData.RepeatPassword)

	if err != nil {
		errorStruct.Message = err.Error()
		JSONResponse(errorStruct, w)
		w.WriteHeader(res)
		return
	}

	salt := GenerateSalt()
	hashedPassword := GenerateSecurePassword(requestData.Password, salt)

	newUser := User{
		Name:     requestData.Name,
		Email:    requestData.Email,
		Password: hashedPassword,
		Salt:     salt,
	}
	db.Save(&newUser)
	w.WriteHeader(http.StatusCreated)
}

func RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	var errorStruct ErrorJSON
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
			errorStruct.Message = err.Error()
			JSONResponse(errorStruct, w)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if !token.Valid {
			errorStruct.Message = err.Error()
			JSONResponse(errorStruct, w)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		email := fmt.Sprintf("%v", claims["email"])

		var oldRefreshToken RefreshToken
		db.Take(&oldRefreshToken, "token = ?", refreshTokenCookie.Value)

		if oldRefreshToken.DeletedAt.Valid {
			db.Delete(&RefreshToken{}, "email = ?", email)

			errorStruct.Message = "token expired"
			JSONResponse(errorStruct, w)
			w.WriteHeader(http.StatusForbidden)

			return
		}

		if exp, ok := claims["exp"].(int64); ok && exp > time.Now().Unix() {
			errorStruct.Message = "token expired"
			JSONResponse(errorStruct, w)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		db.Delete(&oldRefreshToken)

		var user User
		db.Take(&user, "email = ?", email)

		accessToken, _ := MakeTokens(w, user)

		w.WriteHeader(http.StatusAccepted)
		JSONResponse(struct {
			AccessToken string `json:accessToken`
		}{accessToken}, w)

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
	var shops []Shop
	db.Find(&shops)
	JSONResponse(shops, w)
}

func GetShopHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	shopName := params["shop"]

	var shop Shop

	err := db.Where("name = ?", shopName).Take(&shop).Error
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	JSONResponse(&shop, w)
}

func CreateShopHandler(w http.ResponseWriter, r *http.Request) {
	var errorStruct ErrorJSON

	var shop Shop
	err := json.NewDecoder(r.Body).Decode(&shop)

	if err != nil {
		errorStruct.Message = err.Error()
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(errorStruct, w)
		return
	}

	if shop.Name == nil || *shop.Name == "" {
		errorStruct.Message = "name cannot be empty"
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(errorStruct, w)
		return
	}

	err = NameTaken(*shop.Name, &Shop{})
	if err != nil {
		errorStruct.Message = err.Error()
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(errorStruct, w)
		return
	}

	if shop.Description == nil {
		shop.Description = new(string)
	}

	var user User
	db.Take(&user, "email = ?", "a")

	err = db.Where("user_id = ?", user.ID).Take(&Shop{}).Error
	if err == nil {
		errorStruct.Message = "second shop cannot be created"
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(errorStruct, w)
		return
	}

	shop.User = user
	db.Create(&shop)

	w.WriteHeader(http.StatusCreated)
	JSONResponse(shop, w)
}

func UpdateShopHandler(w http.ResponseWriter, r *http.Request) {
	var errorStruct ErrorJSON

	params := mux.Vars(r)
	shopName := params["shop"]

	var requestInfo Shop
	err := json.NewDecoder(r.Body).Decode(&requestInfo)

	if err != nil {
		errorStruct.Message = err.Error()
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(errorStruct, w)
		return
	}

	if requestInfo.Name != nil && *requestInfo.Name == "" {
		errorStruct.Message = "name cannot be empty"
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(errorStruct, w)
		return
	}

	var shop Shop
	db.Where("name = ?", shopName).Take(&shop)

	if requestInfo.Name != nil {
		shop.Name = requestInfo.Name
	}

	if requestInfo.Description != nil {
		shop.Description = requestInfo.Description
	}

	db.Save(&shop)
	w.WriteHeader(http.StatusOK)
}

func DeleteShopHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	shopName := params["shop"]

	err := db.Unscoped().Select("Locations", "Products").Where("name = ?", shopName).Delete(&Shop{}).Error
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

// ===================================================================

// ========================== Products ==============================
func GetProductsHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	requestedCategories := r.Form["category"]

	tx := db

	if len(requestedCategories) > 0 {
		var categoryIDs []uint
		db.Model(&Category{}).Select("id").Where("name IN ?", requestedCategories).Find(&categoryIDs)

		if len(categoryIDs) == 0 {
			w.WriteHeader(http.StatusNotFound)
			JSONResponse(categoryIDs, w)
			return
		}

		var productIDs []uint
		db.Table("product_categories").Select("product_id").Where("category_id IN ?", categoryIDs).Take(&productIDs)

		if len(productIDs) == 0 {
			w.WriteHeader(http.StatusNotFound)
			JSONResponse(productIDs, w)
			return
		}

		tx.Where("id in ?", productIDs)
	}

	var products []Product
	tx.Preload("Categories").Find(&products)

	JSONResponse(products, w)
}

func GetProductHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	productName := params["product"]

	var product Product

	err := db.Preload("Categories").Where("name = ?", productName).Take(&product).Error
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	JSONResponse(product, w)
}

func CreateProductHandler(w http.ResponseWriter, r *http.Request) {
	var errorStruct ErrorJSON

	requestInfo := struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Categories  []int  `json:"categories"`
	}{"", "", nil}

	err := json.NewDecoder(r.Body).Decode(&requestInfo)

	if err != nil {
		errorStruct.Message = err.Error()
		JSONResponse(errorStruct, w)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var categories []Category
	if len(requestInfo.Categories) > 0 {
		db.Find(&categories, requestInfo.Categories)
	}

	if len(requestInfo.Name) <= 0 {
		errorStruct.Message = "product name cannot be empty"
		JSONResponse(errorStruct, w)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var user User
	db.Take(&user, "email = ?", "a")

	var shop Shop
	db.Where("user_id = ?", user.ID).Take(&shop)

	if shop.Name == nil {
		errorStruct.Message = "no shop detected, please create a shop before creating a product"
		JSONResponse(errorStruct, w)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	newProduct := Product{
		Name:        requestInfo.Name,
		Description: requestInfo.Description,
		ShopID:      shop.ID,
		Categories:  categories,
	}

	db.Create(&newProduct)

	w.WriteHeader(http.StatusCreated)
}

func UpdateProductHandler(w http.ResponseWriter, r *http.Request) {
	var errorStruct ErrorJSON

	params := mux.Vars(r)
	productName := params["product"]

	requestInfo := struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Categories  []int   `json:"categories"`
	}{nil, nil, nil}

	err := json.NewDecoder(r.Body).Decode(&requestInfo)

	if err != nil {
		errorStruct.Message = err.Error()
		JSONResponse(errorStruct, w)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var product Product
	db.Preload("Categories").Where("name = ?", productName).Take(&product)

	if requestInfo.Name != nil {
		product.Name = *requestInfo.Name
	}

	if requestInfo.Description != nil {
		product.Description = *requestInfo.Description
	}

	if requestInfo.Categories != nil {
		var categories []Category
		db.Find(&categories, requestInfo.Categories)

		product.Categories = categories
	}

	db.Save(&product)
}

func DeleteProductHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	productName := params["product"]

	db.Unscoped().Where("name = ?", productName).Delete(&Product{})
}

// ===================================================================

// ========================== Locations ==============================
func GetLocationsHandler(w http.ResponseWriter, r *http.Request) {
	var errorStruct ErrorJSON

	params := mux.Vars(r)
	shopName := params["shop"]

	var shop Shop
	db.Where("name = ?", shopName).Find(&shop)

	if shop.Name == nil {
		errorStruct.Message = "shop not found"
		JSONResponse(errorStruct, w)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var locations []Location
	db.Where("shop_id = ?", shop.ID).Find(&locations)

	JSONResponse(locations, w)
}

func GetLocationHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	locationID := params["locationid"]

	var location Location

	if db.Take(&location, locationID).Error != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	JSONResponse(&location, w)
}

func CreateLocationHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	shopName := params["shop"]

	var requestInfo Location
	err := json.NewDecoder(r.Body).Decode(&requestInfo)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(ErrorJSON{
			Message: err.Error(),
		}, w)
		return
	}

	var shop Shop
	db.Where("name = ?", shopName).Take(&shop)

	requestInfo.Shop = shop
	db.Create(&requestInfo)

	w.WriteHeader(http.StatusCreated)
}
func DeleteLocationsHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	shopName := params["shop"]

	var shop Shop
	err := db.Where("name = ?", shopName).Take(&shop).Error

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}

	err = db.Where("shop_id = ?", shop.ID).Delete(&Location{}).Error
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}

	w.WriteHeader(http.StatusOK)
}

func UpdateLocationHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	locationID := params["locationid"]

	var requestInfo Location
	err := json.NewDecoder(r.Body).Decode(&requestInfo)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(ErrorJSON{
			Message: err.Error(),
		}, w)
		return
	}
	// tx :=
	db.Model(&Location{}).Where("id = ?", locationID)

	// if requestInfo.Coordinates != "" {
	// 	tx.Updates(Location{Coordinates: requestInfo.Coordinates})
	// }

	w.WriteHeader(http.StatusOK)
}

func DeleteLocationHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	locationID := params["locationid"]

	db.Delete(&Location{}, locationID)
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
	caregoryID := params["categoryid"]

	var category Category

	err := db.Take(&category, caregoryID).Error
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	JSONResponse(&category, w)
}

func CreateCategoryHandler(w http.ResponseWriter, r *http.Request) {
	var errorStruct ErrorJSON

	r.ParseMultipartForm(10 << 20)

	var requestInfo Category

	requestInfo.File = FileUpload(r, "file", "category-*.png")
	*requestInfo.Name = r.FormValue("name")

	if requestInfo.Name == nil {
		errorStruct.Message = "category name cannot be empty"
		JSONResponse(errorStruct, w)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	db.Create(&requestInfo)

	w.WriteHeader(http.StatusCreated)
}

func UpdateCategoryHandler(w http.ResponseWriter, r *http.Request) {
	var errorStruct ErrorJSON

	r.ParseMultipartForm(10 << 20)

	params := mux.Vars(r)
	categoryID := params["categoryid"]

	var category Category

	err := db.Take(&category, categoryID).Error
	if err != nil {
		errorStruct.Message = "category not found"
		JSONResponse(errorStruct, w)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	image := FileUpload(r, "file", "category-*.png")
	name := r.FormValue("name")

	if len(name) > 0 {
		*category.Name = name
	}

	if len(image) > 0 {
		category.File = image
	}

	db.Save(&category)
}

func DeleteCategoryHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	caregoryID := params["categoryid"]

	db.Unscoped().Delete(&Category{}, caregoryID)
}
