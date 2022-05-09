package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/pbkdf2"
)

type permissionValidator func(string) bool

// =========================== Handlers ===================================
func Login(w http.ResponseWriter, r *http.Request) {
	errorMSG := "blogi duomenys"
	//Creates a struct used to store data decoded from the body
	requestData := struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}{"", ""}

	json.NewDecoder(r.Body).Decode(&requestData)

	var userDatabaseData User

	// Finds user by email in database, if no user, then returns "bad request"
	err := db.Take(&userDatabaseData, "name = ? and temporary = ?", requestData.Name, false).Error
	if err != nil {
		Response(w, http.StatusBadRequest, errorMSG)
		return
	}

	hashedPassword := GenerateSecurePassword(requestData.Password, userDatabaseData.Salt)
	//checks if salted hashed password from database matches the sent in salted hashed password
	if hashedPassword != userDatabaseData.Password {
		Response(w, http.StatusBadRequest, errorMSG)
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

func CheckEmail(w http.ResponseWriter, r *http.Request) {
	request := struct {
		Email string `json:"email"`
	}{""}

	err := json.NewDecoder(r.Body).Decode(&request)

	if err != nil {
		Response(w, http.StatusBadRequest, "blogas duomenų formatas")
		return
	}

	err = CheckEmailAvailability(request.Email)
	if err != nil {
		Response(w, http.StatusConflict, "šis el.pašto adresas jau užimtas")
	}
}

//CreateAccount decodes user sent in data, verifies that
//it is formatted correctly, and tries to create an account in
//the database
func CreateAccount(w http.ResponseWriter, r *http.Request) {
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
		Response(w, http.StatusBadRequest, "blogas duomenų formatas")
		return
	}

	res, err := PerformUserDataChecks(requestData.Name, requestData.Email, requestData.Password, requestData.RepeatPassword)

	if err != nil {
		Response(w, res, err.Error())
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

	if err = db.Save(&newUser).Error; err != nil {
		Response(w, http.StatusInternalServerError, "klaida saugojant duomenis. bandykite dar kartą")
		return
	}

	accessToken, _ := MakeTokens(w, newUser)

	w.WriteHeader(http.StatusCreated)
	JSONResponse(struct {
		AccessToken string `json:accessToken`
	}{accessToken}, w)
}

func RefreshTokens(w http.ResponseWriter, r *http.Request) {
	refreshTokenCookie, err := r.Cookie("Refresh-Token")

	if err == nil {
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(refreshTokenCookie.Value, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("įvyko klaida. bandykite dar kartą")
			}
			return signKey, nil
		})

		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if !token.Valid {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		email := fmt.Sprintf("%v", claims["email"])

		var oldRefreshToken RefreshToken
		db.Take(&oldRefreshToken, "token = ?", refreshTokenCookie.Value)

		if oldRefreshToken.DeletedAt.Valid {
			db.Delete(&RefreshToken{}, "email = ?", email)

			Response(w, http.StatusForbidden, "žetono galiojimo laikas pasibaigęs")
			return
		}

		if exp, ok := claims["exp"].(int64); ok && exp > time.Now().Unix() {
			Response(w, http.StatusUnauthorized, "žetono galiojimo laikas pasibaigęs")
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
}

// ===================================================================

// ============================= Helpers =============================

func CheckEmailAvailability(email string) error {
	//if no record of the email is found, returns an error
	err := db.Take(&User{}, "email = ?", email).Error
	if err == nil {
		return errors.New("šis el.pašto adresas jau užimtas")
	}

	return nil
}

//CreateNewAccount creates an account if the sent data
//is correctly formatted
func PerformUserDataChecks(name string, email string, password string, repeatedPassword string) (httpStatus int, err error) {
	if len(name) == 0 {
		return http.StatusBadRequest, errors.New("vardas yra privalomas")
	}

	err = NameTaken(name, &User{})
	if err != nil {
		return http.StatusConflict, err
	}

	if !emailRegex.MatchString(email) {
		return http.StatusBadRequest, errors.New("blogas el.pašto formatas")
	}

	err = CheckEmailAvailability(email)
	if err != nil {
		return http.StatusConflict, err
	}

	err = CheckIfPasswordValid(password, repeatedPassword)
	if err != nil {
		return http.StatusBadRequest, err
	}

	return http.StatusOK, nil
}

//checks that, while registering a new account that
//the provided password matches the repeated password, is atleast 8 characters long and
//contains at least one number and one capital letter
func CheckIfPasswordValid(passwordOne string, passwordTwo string) error {
	if passwordOne != passwordTwo {
		return errors.New("slaptažodžiai nesutampa")
	}

	// if len(passwordOne) < 8 {
	// 	return errors.New("slaptažodį turi sudaryti bent 8 simboliai")
	// }

	// if !passwordRegex.MatchString(passwordOne) {
	// 	return errors.New("slaptažodį turi sudaryti bent vienas skaičius ir bent viena didžioji raidė")
	// }

	return nil
}

func MakeTokens(w http.ResponseWriter, user User) (string, string) {
	claims := map[string]interface{}{
		"name":        user.Name,
		"email":       user.Email,
		"permissions": user.Permissions,
		"isSet":       true, // For frontend
		"shop":        user.ShopCodename,
		"exp":         time.Now().Add(time.Hour).Unix(),
	}
	accessToken, _ := GenerateToken(claims)
	// http.SetCookie(w, &http.Cookie{Name: "Access-Token", Value: accessToken, MaxAge: 60, SameSite: http.SameSiteNoneMode, Secure: true})
	http.SetCookie(w, &http.Cookie{Name: "Access-Token", Value: accessToken, MaxAge: 60})

	claims["exp"] = time.Now().Add(time.Hour * 24 * 7).Unix()
	refreshToken, _ := GenerateToken(claims)

	refreshDatabaseEntry := RefreshToken{
		Token: refreshToken,
		Email: user.Email,
	}
	db.Create(&refreshDatabaseEntry)

	// http.SetCookie(w, &http.Cookie{Name: "Refresh-Token", Value: refreshToken, HttpOnly: true, MaxAge: 60 * 60 * 24 * 7, SameSite: http.SameSiteNoneMode, Secure: true})
	http.SetCookie(w, &http.Cookie{Name: "Refresh-Token", Value: refreshToken, HttpOnly: true, MaxAge: 60 * 60 * 24 * 7})

	return accessToken, refreshToken
}

//GenerateSalt creates a pseudorandom salt used in password salting
func GenerateSalt() string {
	salt, _ := uuid.NewRandom()
	return salt.String()
}

//GenerateSecurePassword generates a password using PBKDF2 standard
func GenerateSecurePassword(password string, salt string) string {
	hashedPassword := pbkdf2.Key([]byte(password), []byte(salt), 4096, 32, sha1.New)

	return hex.EncodeToString(hashedPassword)
}

func isCourier(next http.HandlerFunc) http.HandlerFunc {
	return CheckPermissions(next, HasCourierPermissions)
}

func isAdmin(next http.HandlerFunc) http.HandlerFunc {
	return CheckPermissions(next, HasAdminPermissions)
}

func isFarmer(next http.HandlerFunc) http.HandlerFunc {
	return CheckPermissions(next, HasFarmerPermissions)
}

func CheckPermissions(next http.HandlerFunc, hasPermissions permissionValidator) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		permissions := GetClaim("permissions", r)

		if permissions != nil && hasPermissions(*permissions) {
			next.ServeHTTP(w, r)
			return
		}

		w.WriteHeader(http.StatusUnauthorized)
	})
}

func isProductOwner(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		productName := params["product"]

		var product Product
		err := db.Preload("Shop").Preload("Shop.User").First(&product, "codename = ?", productName).Error

		if err != nil {
			Response(w, http.StatusBadRequest, "produkto rasti nepavyko")
			return
		}

		email := GetClaim("email", r)

		if email == nil || product.Shop.User.Email != *email {
			Response(w, http.StatusUnauthorized, "jūs negalite koreguoti šio produkto")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func WithContext(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := jwt.MapClaims{}
		accessTokenCookie, err := r.Cookie("Access-Token")
		if err == nil {
			jwt.ParseWithClaims(accessTokenCookie.Value, claims, func(token *jwt.Token) (interface{}, error) {
				_, ok := token.Method.(*jwt.SigningMethodHMAC)
				if !ok {
					return nil, fmt.Errorf("įvyko klaida, bandykite dar kartą")
				}
				return signKey, nil
			})
		}

		ctx := context.WithValue(r.Context(), ctxKey{}, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isAuthorized(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accessTokenCookie, err := r.Cookie("Access-Token")
		if err == nil {
			claims := jwt.MapClaims{}
			token, err := jwt.ParseWithClaims(accessTokenCookie.Value, claims, func(token *jwt.Token) (interface{}, error) {
				_, ok := token.Method.(*jwt.SigningMethodHMAC)
				if !ok {
					return nil, fmt.Errorf("įvyko klaida, bandykite dar kartą")
				}
				return signKey, nil
			})

			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			if token.Valid {
				ctx := context.WithValue(r.Context(), ctxKey{}, claims)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		w.WriteHeader(http.StatusUnauthorized)
	})
}

func GenerateToken(claimsMap map[string]interface{}) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)
	for k, v := range claimsMap {
		claims[k] = v
	}

	tokenString, err := token.SignedString(signKey)

	if err != nil {
		return "", err
	}

	return tokenString, nil
}
