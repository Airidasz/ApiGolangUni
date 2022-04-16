package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/shopspring/decimal"
	"gorm.io/gorm/clause"
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
func Login(w http.ResponseWriter, r *http.Request) {
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

func Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "Refresh-Token", Value: "", MaxAge: -1, SameSite: http.SameSiteNoneMode, Secure: true})
	http.SetCookie(w, &http.Cookie{Name: "Access-Token", Value: "", MaxAge: -1, SameSite: http.SameSiteNoneMode, Secure: true})

	w.WriteHeader(http.StatusAccepted)
}

//CreateAccount decodes user sent in data, verifies that
//it is formatted correctly, and tries to create an account in
//the database
func CreateAccount(w http.ResponseWriter, r *http.Request) {
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

	accessToken, _ := MakeTokens(w, newUser)

	w.WriteHeader(http.StatusCreated)
	JSONResponse(struct {
		AccessToken string `json:accessToken`
	}{accessToken}, w)
}

func RefreshTokens(w http.ResponseWriter, r *http.Request) {
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
func GetShops(w http.ResponseWriter, r *http.Request) {
	var shops []Shop
	db.Find(&shops)
	JSONResponse(shops, w)
}

func GetShopOrders(w http.ResponseWriter, r *http.Request) {
	email := GetClaim("email", r)

	var shop Shop
	GetShopByEmail(email, &shop, false, "id")

	var orderedProducts []OrderedProduct

	db.Preload(clause.Associations).Joins("left join products on products.id  = ordered_products.product_id").Where("products.shop_id = ?", shop.ID).Find(&orderedProducts)
	JSONResponse(orderedProducts, w)
}

func GetShop(w http.ResponseWriter, r *http.Request) {
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

func CreateShop(w http.ResponseWriter, r *http.Request) {
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

func UpdateShop(w http.ResponseWriter, r *http.Request) {
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

func DeleteShop(w http.ResponseWriter, r *http.Request) {
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

	tx.Find(&products)

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

func CreateProduct(w http.ResponseWriter, r *http.Request) {
	var errorStruct ErrorJSON

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

	r.ParseMultipartForm(10 << 20)

	image := FileUpload(r, "file", "product-*.png")

	var newProduct Product

	decoder := schema.NewDecoder()
	decoder.Decode(&newProduct, r.Form)

	if newProduct.Name == nil || len(*newProduct.Name) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "product name cannot be empty"
		JSONResponse(errorStruct, w)
		return
	}

	if newProduct.Quantity < 0 {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "quantity cannot be less than zero"
		JSONResponse(errorStruct, w)
		return
	}

	if newProduct.Amount.IsNegative() {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "price cannot be less than zero"
		JSONResponse(errorStruct, w)
		return
	}

	// Get category objects from IDs
	categoriesJson := r.FormValue("categories")
	categories, _ := ParseCategories(categoriesJson)

	// Add additional information
	newProduct.Codename = GenerateCodename(*newProduct.Name, true)
	newProduct.Image = image
	newProduct.ShopID = shop.ID
	newProduct.Categories = categories

	// Create
	db.Create(&newProduct)

	w.WriteHeader(http.StatusCreated)
	JSONResponse(newProduct, w)
}

func UpdateProduct(w http.ResponseWriter, r *http.Request) {
	var errorStruct ErrorJSON

	params := mux.Vars(r)
	productName := params["product"]

	r.ParseMultipartForm(10 << 20)
	image := FileUpload(r, "file", "product-*.png")

	requestInfo := struct {
		Name        *string          `json:"name"`
		Description *string          `json:"description"`
		Categories  []string         `json:"categories"`
		Amount      *decimal.Decimal `json:"amount"`
		Public      *bool            `json:"public"`
		Quantity    *int             `json:"quantity"`
	}{}

	decoder := schema.NewDecoder()
	err := decoder.Decode(&requestInfo, r.Form)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = err.Error()
		JSONResponse(errorStruct, w)
		return
	}

	var product Product
	db.Preload("Categories").Where("codename = ?", productName).Take(&product)

	if requestInfo.Name != nil && *product.Name != *requestInfo.Name {
		product.Name = requestInfo.Name
		product.Codename = GenerateCodename(*requestInfo.Name, true)
	}

	if requestInfo.Description != nil && *product.Description != *requestInfo.Description {
		product.Description = requestInfo.Description
	}

	if requestInfo.Categories != nil {
		var categories []Category
		db.Find(&categories, "id in ?", requestInfo.Categories)
		product.Categories = categories
	}

	if len(image) > 0 {
		product.Image = image
	}

	db.Save(&product)
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

	requestInfo := struct {
		Name        *string          `json:"name"`
		Description *string          `json:"description"`
		Categories  []string         `json:"categories"`
		Amount      *decimal.Decimal `json:"amount"`
		Public      *bool            `json:"public"`
		Quantity    *int             `json:"quantity"`
	}{}

	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	err := decoder.Decode(&requestInfo, r.Form)

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
	if !isEdit && (requestInfo.Name == nil || len(*requestInfo.Name) == 0) {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "product name cannot be empty"
		JSONResponse(errorStruct, w)
		return
	}

	if requestInfo.Name != nil {
		product.Name = requestInfo.Name
		product.Codename = GenerateCodename(*requestInfo.Name, true)
	}

	// Quantity
	if requestInfo.Quantity != nil && *requestInfo.Quantity < 0 {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "quantity cannot be less than zero"
		JSONResponse(errorStruct, w)
		return
	}

	if requestInfo.Quantity != nil {
		product.Quantity = *requestInfo.Quantity
	}

	// Amount
	if requestInfo.Amount != nil && requestInfo.Amount.IsNegative() {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "price cannot be less than zero"
		JSONResponse(errorStruct, w)
		return
	}

	if requestInfo.Amount != nil {
		product.Amount = *requestInfo.Amount
	}

	// Description
	if requestInfo.Description != nil {
		product.Description = requestInfo.Description
	}

	// Public product
	if requestInfo.Public != nil {
		product.Public = *requestInfo.Public
	}

	// Categories
	if requestInfo.Categories != nil {
		var categories []Category
		db.Find(&categories, "id in ?", requestInfo.Categories)
		product.Categories = categories
	}

	// Image
	if len(image) > 0 {
		product.Image = image
	}

	// Add or save to Database
	if isEdit {
		db.Save(&product)
	} else {
		db.Create(&product)
	}

	JSONResponse(product, w)
}

func DeleteProduct(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	productName := params["product"]

	db.Unscoped().Where("codename = ?", productName).Delete(&Product{})
}

// ===================================================================

// ========================== Locations ==============================
func GetLocations(w http.ResponseWriter, r *http.Request) {
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

func GetLocation(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	locationID := params["locationid"]

	var location Location

	if db.Take(&location, locationID).Error != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	JSONResponse(&location, w)
}

func CreateLocation(w http.ResponseWriter, r *http.Request) {
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
func DeleteLocations(w http.ResponseWriter, r *http.Request) {
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

func UpdateLocation(w http.ResponseWriter, r *http.Request) {
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

func DeleteLocation(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	locationID := params["locationid"]

	db.Delete(&Location{}, locationID)
}

// ===================================================================

// ========================== Categories ==============================
func GetCategories(w http.ResponseWriter, r *http.Request) {
	var categories []Category
	db.Find(&categories)
	JSONResponse(categories, w)
}

func GetCategory(w http.ResponseWriter, r *http.Request) {
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

func CreateCategory(w http.ResponseWriter, r *http.Request) {
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

func UpdateCategory(w http.ResponseWriter, r *http.Request) {
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

func DeleteCategory(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	caregoryID := params["categoryid"]

	db.Unscoped().Delete(&Category{}, "id = ?", caregoryID)
}

// ===================================================================

// ========================== Orders ==============================

func PlaceOrder(w http.ResponseWriter, r *http.Request) {
	var errorStruct ErrorJSON

	// Check if the user is not a farmer
	permissions := GetClaim("permissions", r)

	if strings.Contains(strings.ToLower(permissions), "f") {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "you cannot place an order with a farmer account"
		JSONResponse(errorStruct, w)
		return
	}

	// Decode request
	request := struct {
		Email           string           `json:"email"`
		Note            string           `json:"note"`
		Shipping        string           `json:"shipping"`
		Payment         string           `json:"payment"`
		OrderedProducts []OrderedProduct `json:"orderedProducts"`
	}{}

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorStruct.Message = "unable to parse body to json"
		JSONResponse(errorStruct, w)
		return
	}

	// Check for errors:
	// does product exist, is the quantity correct
	var orderedProducts []OrderedProduct
	errors := make(map[string]string)

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
			ProductID: product.ID,
			Quantity:  orderedProduct.Quantity,
		}

		orderedProducts = append(orderedProducts, newProductOrder)
	}

	if len(errors) > 0 {
		fmt.Println(errors)
		return /// return error response
	}

	// No errors, create order
	order := Order{
		Email:    request.Email,
		Status:   1,
		Note:     request.Note,
		Shipping: request.Shipping,
		Payment:  request.Payment,
	}

	db.Create(&order)

	for _, orderedProduct := range orderedProducts {
		orderedProduct.OrderID = order.ID
		db.Create(&orderedProduct)
	}

	w.WriteHeader(http.StatusCreated)
}
