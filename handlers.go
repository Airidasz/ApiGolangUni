package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
	"gorm.io/gorm/clause"
)

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
		JSONResponse(struct{}{}, w)
		return
	}

	hashedPassword := GenerateSecurePassword(requestData.Password, userDatabaseData.Salt)
	//checks if salted hashed password from database matches the sent in salted hashed password
	if hashedPassword != userDatabaseData.Password {
		w.WriteHeader(http.StatusUnauthorized)
		JSONResponse(struct{}{}, w)
		return
	}

	MakeTokens(w, requestData.Email)

	w.WriteHeader(http.StatusAccepted)
	JSONResponse(struct{}{}, w)
}

//RegisterPageHandler decodes user sent in data, verifies that
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
		JSONResponse(struct{}{}, w)
		return
	}

	err = CheckIfPasswordValid(requestData.Password, requestData.RepeatPassword)
	if err != nil {
		JSONResponse(struct{}{}, w)
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
			fmt.Fprint(w, err.Error())
			return
		}

		if !token.Valid {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, err.Error())
			return
		}

		email := fmt.Sprintf("%v", claims["email"])

		var oldRefreshToken RefreshToken
		db.Take(&oldRefreshToken, "token = ?", refreshTokenCookie.Value)

		if oldRefreshToken.DeletedAt.Valid {
			db.Delete(&RefreshToken{}, "email = ?", email)
			w.WriteHeader(http.StatusForbidden)
			return
		}

		if exp, ok := claims["exp"].(int64); ok && exp > time.Now().Unix() {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		db.Delete(&oldRefreshToken)

		MakeTokens(w, email)
		w.WriteHeader(http.StatusAccepted)
		return
	}

	w.WriteHeader(http.StatusUnauthorized)
	JSONResponse(struct{}{}, w)
}

func GetShopsHandler(w http.ResponseWriter, r *http.Request) {
	var shops []Shop
	db.Find(&shops)
	JSONResponse(shops, w)
}

func GetShopHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	shopID, err := strconv.Atoi(params["shopid"])

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(struct{}{}, w)
		return
	}

	var shop Shop
	db.Preload(clause.Associations).Take(&shop, shopID)
	JSONResponse(&shop, w)
}

func GetLocationsHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	shopID, err := strconv.Atoi(params["shopid"])

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(struct{}{}, w)
		return
	}

	var locations []Location
	db.Where("shop_id = ?", shopID).Find(&locations)
	JSONResponse(locations, w)
}

func GetProductsHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	shopID, err := strconv.Atoi(params["shopid"])

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(struct{}{}, w)
		return
	}

	var products []Product
	db.Where("shop_id = ?", shopID).Find(&products)
	JSONResponse(products, w)
}

func CreateShopHandler(w http.ResponseWriter, r *http.Request) {
	email := fmt.Sprintf("%v", r.Context().Value(ctxKey{}))

	var user User
	db.Take(&user, "email = ?", email)

	var requestInfo Shop
	err := json.NewDecoder(r.Body).Decode(&requestInfo)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSONResponse(struct{}{}, w)
		return
	}

	requestInfo.User = user
	db.Create(&requestInfo)

	w.WriteHeader(http.StatusCreated)
	JSONResponse(struct{}{}, w)
}

//checks that, while registering a new account that
//the provided password matches the repeated password, is atleast 8 characters long and
//contains at least one number and one capital letter
func CheckIfPasswordValid(passwordOne string, passwordTwo string) error {
	if passwordOne != passwordTwo {
		return errors.New("passwords do not match")
	}

	// if len(passwordOne) < 8 {
	// 	return errors.New("passwords too short")
	// }

	// if !passwordRegex.MatchString(passwordOne) {
	// 	return errors.New("passwords needs to contain at least one number and one capital letter")
	// }

	return nil
}

func LandingPage(w http.ResponseWriter, r *http.Request) {
	b := struct {
		Make    string `json:"make"`
		Model   string `json:"model"`
		Mileage int    `json:"mileage"`
	}{"Ford", "Taurus", 2000010}

	JSONResponse(b, w)
}
