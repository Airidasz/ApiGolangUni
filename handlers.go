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
		Name     string `json:"name"`
		Password string `json:"password"`
	}{"", ""}

	json.NewDecoder(r.Body).Decode(&requestData)

	var userDatabaseData User

	// Finds user by email in database, if no user, then returns "bad request"
	err := db.Take(&userDatabaseData, "name = ?", requestData.Name).Error
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(errorStruct, w)
		return
	}

	hashedPassword := GenerateSecurePassword(requestData.Password, userDatabaseData.Salt)
	//checks if salted hashed password from database matches the sent in salted hashed password
	if hashedPassword != userDatabaseData.Password {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(errorStruct, w)
		return
	}
	accessToken, _ := MakeTokens(w, userDatabaseData)

	w.WriteHeader(http.StatusAccepted)
	JSONResponse(struct {
		AccessToken string `json:accessToken`
	}{accessToken}, w)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "Refresh-Token", Value: "", MaxAge: -1, SameSite: http.SameSiteNoneMode, Secure: true})
	http.SetCookie(w, &http.Cookie{Name: "Access-Token", Value: "", MaxAge: -1, SameSite: http.SameSiteNoneMode, Secure: true})

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
		Farmer         bool   `json:"farmer"`
	}{"", "", "", "", false}

	err := json.NewDecoder(r.Body).Decode(&requestData)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "unable to parse body to json"
		JSONResponse(errorStruct, w)
		return
	}

	res, err := PerformUserDataChecks(requestData.Name, requestData.Email, requestData.Password, requestData.RepeatPassword)

	if err != nil {
		w.WriteHeader(res)
		errorStruct.Message = err.Error()
		JSONResponse(errorStruct, w)
		return
	}

	salt := GenerateSalt()
	hashedPassword := GenerateSecurePassword(requestData.Password, salt)

	permissions := ""
	if requestData.Farmer {
		permissions += "f"
	}

	newUser := User{
		Name:        requestData.Name,
		Email:       requestData.Email,
		Password:    hashedPassword,
		Permissions: permissions,
		Salt:        salt,
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
			w.WriteHeader(http.StatusUnauthorized)
			errorStruct.Message = err.Error()
			JSONResponse(errorStruct, w)
			return
		}

		if !token.Valid {
			w.WriteHeader(http.StatusUnauthorized)
			errorStruct.Message = err.Error()
			JSONResponse(errorStruct, w)
			return
		}

		email := fmt.Sprintf("%v", claims["email"])

		var oldRefreshToken RefreshToken
		db.Take(&oldRefreshToken, "token = ?", refreshTokenCookie.Value)

		if oldRefreshToken.DeletedAt.Valid {
			db.Delete(&RefreshToken{}, "email = ?", email)

			w.WriteHeader(http.StatusForbidden)
			errorStruct.Message = "token expired"
			JSONResponse(errorStruct, w)

			return
		}

		if exp, ok := claims["exp"].(int64); ok && exp > time.Now().Unix() {
			w.WriteHeader(http.StatusUnauthorized)
			errorStruct.Message = "token expired"
			JSONResponse(errorStruct, w)
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
	errorStruct.Message = "unauthorized"
	JSONResponse(errorStruct, w)
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

	err := db.Where("codename = ?", shopName).Take(&shop).Error
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	JSONResponse(&shop, w)
}

func CreateShopHandler(w http.ResponseWriter, r *http.Request) {
	var errorStruct ErrorJSON

	email := GetClaim("email", r)
	err := GetShopByEmail(email, &Shop{}, false)
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
	db.Create(&shop)

	// Update user info
	user.ShopCodename = &shop.Codename
	db.Save(&user)

	// Send tokens with correct info
	MakeTokens(w, user)

	w.WriteHeader(http.StatusCreated)
	JSONResponse(shop, w)
}

func UpdateShopHandler(w http.ResponseWriter, r *http.Request) {
	var errorStruct ErrorJSON

	// Check if user has a shop
	email := GetClaim("email", r)

	var shop Shop
	err := GetShopByEmail(email, &shop, false)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "shop not found, please create a shop"
		JSONResponse(errorStruct, w)
		return
	}

	var requestInfo Shop
	err = json.NewDecoder(r.Body).Decode(&requestInfo)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = err.Error()
		JSONResponse(errorStruct, w)
		return
	}

	if requestInfo.Name != nil && *requestInfo.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "name cannot be empty"
		JSONResponse(errorStruct, w)
		return
	}

	if requestInfo.Name != nil {
		shop.Name = requestInfo.Name
		shop.Codename = GenerateCodename(*shop.Name, false)

		var user User
		db.Take(&user, "id = ?", shop.UserID)

		user.ShopCodename = &shop.Codename
		db.Save(&user)
		// Send tokens with correct info
		MakeTokens(w, user)
	}

	if requestInfo.Description != nil {
		shop.Description = requestInfo.Description
	}

	db.Save(&shop)

	JSONResponse(shop, w)
}

func DeleteShopHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	shopName := params["shop"]

	err := db.Unscoped().Select("Locations", "Products").Where("codename = ?", shopName).Delete(&Shop{}).Error
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

// ===================================================================

// ========================== Products ==============================
func GetProductsHandler(w http.ResponseWriter, r *http.Request) {
	var products []Product

	r.ParseForm()
	requestedCategories := r.Form["category"]

	tx := db.Preload("Categories")

	if len(requestedCategories) > 0 {
		var categoryIDs []string
		db.Model(&Category{}).Select("id").Where("name IN ?", requestedCategories).Find(&categoryIDs)

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

	tx.Find(&products)

	JSONResponse(products, w)
}

func GetProductHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	productName := params["product"]

	var product Product

	err := db.Preload("Categories").Where("codename = ?", productName).Take(&product).Error
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	JSONResponse(product, w)
}

func CreateProductHandler(w http.ResponseWriter, r *http.Request) {
	var errorStruct ErrorJSON
	r.ParseMultipartForm(10 << 20)

	image := FileUpload(r, "file", "product-*.png")
	name := r.FormValue("name")
	description := r.FormValue("description")
	categoriesJson := r.FormValue("categories")

	var categoryIDs []string
	err := json.Unmarshal([]byte(categoriesJson), &categoryIDs)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = err.Error()
		JSONResponse(errorStruct, w)
		return
	}

	var categories []Category
	if len(categoryIDs) > 0 {
		db.Find(&categories, "id in ?", categoryIDs)
	}

	if len(name) <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "product name cannot be empty"
		JSONResponse(errorStruct, w)
		return
	}

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

	newProduct := Product{
		Name:        name,
		Codename:    GenerateCodename(name, true),
		Description: description,
		Image:       image,
		ShopID:      shop.ID,
		Categories:  categories,
	}

	db.Create(&newProduct)

	w.WriteHeader(http.StatusCreated)
	JSONResponse(newProduct, w)
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
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = err.Error()
		JSONResponse(errorStruct, w)
		return
	}

	var product Product
	db.Preload("Categories").Where("codename = ?", productName).Take(&product)

	if requestInfo.Name != nil && product.Name != *requestInfo.Name {
		product.Name = *requestInfo.Name
		product.Codename = GenerateCodename(*requestInfo.Name, true)
	}

	if requestInfo.Description != nil && product.Description != *requestInfo.Description {
		product.Description = *requestInfo.Description
	}

	if requestInfo.Categories != nil {
		var categories []Category
		db.Find(&categories, requestInfo.Categories)

		product.Categories = categories
	}

	db.Save(&product)
	JSONResponse(product, w)
}

func DeleteProductHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	productName := params["product"]

	db.Unscoped().Where("codename = ?", productName).Delete(&Product{})
}

// ===================================================================

// ========================== Locations ==============================
func GetLocationsHandler(w http.ResponseWriter, r *http.Request) {
	var errorStruct ErrorJSON

	params := mux.Vars(r)
	shopName := params["shop"]

	var shop Shop
	db.Where("codename = ?", shopName).Find(&shop)

	if shop.Name == nil {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "shop not found"
		JSONResponse(errorStruct, w)
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
	var errorStruct ErrorJSON
	// Block user from creating a second shop
	email := GetClaim("email", r)

	var shop Shop
	err := GetShopByEmail(email, &shop, false, "id")

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "please create a shop first"
		JSONResponse(errorStruct, w)
		return
	}

	var requestInfo Location
	err = json.NewDecoder(r.Body).Decode(&requestInfo)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(ErrorJSON{
			Message: err.Error(),
		}, w)
		return
	}

	// requestInfo.ShopCodename = shop.Codename
	db.Create(&requestInfo)

	w.WriteHeader(http.StatusCreated)
}
func DeleteLocationsHandler(w http.ResponseWriter, r *http.Request) {
	var errorStruct ErrorJSON

	email := GetClaim("email", r)

	var shop Shop
	err := GetShopByEmail(email, &shop, false, "id")

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "please create a shop first"
		JSONResponse(errorStruct, w)
		return
	}

	db.Where("shop_id = ?", shop.ID).Delete(&Location{})
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
	name := r.FormValue("name")
	requestInfo.Name = &name
	requestInfo.Codename = GenerateCodename(name, false)

	if requestInfo.Name == nil {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "category name cannot be empty"
		JSONResponse(errorStruct, w)
		return
	}

	db.Create(&requestInfo)

	w.WriteHeader(http.StatusCreated)
}

func UpdateCategoryHandler(w http.ResponseWriter, r *http.Request) {
	var errorStruct ErrorJSON

	params := mux.Vars(r)
	categoryID := params["categoryid"]

	r.ParseMultipartForm(10 << 20)

	var category Category

	err := db.Take(&category, "id = ?", categoryID).Error
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "category not found"
		JSONResponse(errorStruct, w)
		return
	}

	image := FileUpload(r, "file", "category-*.png")
	name := r.FormValue("name")

	if len(name) > 0 && *category.Name != name {
		*category.Name = name
		category.Codename = GenerateCodename(name, false)
	}

	if len(image) > 0 {
		category.File = image
	}

	db.Save(&category)
}

func DeleteCategoryHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	caregoryID := params["categoryid"]

	db.Unscoped().Delete(&Category{}, "id = ?", caregoryID)
}
