package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/pbkdf2"
	"gorm.io/gorm/clause"
)

func CheckEmailAvailability(email string) error {
	//if no record of the email is found, returns an error
	err := db.Take(&User{}, "email = ?", email).Error
	if err == nil {
		return errors.New("this email is taken")
	}

	return nil
}

//CreateNewAccount creates an account if the sent data
//is correctly formatted
func PerformUserDataChecks(name string, email string, password string, repeatedPassword string) (httpStatus int, err error) {
	if len(name) == 0 {
		return http.StatusNotAcceptable, errors.New("username cannot be empty")
	}

	err = NameTaken(name, &User{})
	if err != nil {
		return http.StatusNotAcceptable, err
	}

	if !emailRegex.MatchString(email) {
		return http.StatusNotAcceptable, errors.New("please enter a valid email address")
	}

	err = CheckEmailAvailability(email)
	if err != nil {
		return http.StatusNotAcceptable, err
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

func MakeTokens(w http.ResponseWriter, user User) (string, string) {
	claims := map[string]interface{}{
		"id":          user.ID,
		"email":       user.Email,
		"permissions": user.Permissions,
		"shop":        user.ShopCodename,
		"exp":         time.Now().Add(time.Hour).Unix(),
	}
	accessToken, _ := GenerateToken(claims)
	// http.SetCookie(w, &http.Cookie{Name: "Access-Token", Value: accessToken, MaxAge: 60, SameSite: http.SameSiteNoneMode, Secure: true})
	http.SetCookie(w, &http.Cookie{Name: "Access-Token", Value: accessToken, MaxAge: 60000})

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

func isAdmin(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accessTokenCookie, err := r.Cookie("Access-Token")
		if err == nil {
			claims := jwt.MapClaims{}
			token, err := jwt.ParseWithClaims(accessTokenCookie.Value, claims, func(token *jwt.Token) (interface{}, error) {
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

			if token.Valid {
				permissions := claims["permissions"].(string)

				if hasAdminPermissions(permissions) {
					ctx := context.WithValue(r.Context(), ctxKey{}, claims)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
		}

		w.WriteHeader(http.StatusUnauthorized)
		JSONResponse(ErrorJSON{
			Message: "unauthorized",
		}, w)

	})
}

func hasAdminPermissions(permissions string) bool {
	return strings.ContainsAny(permissions, "aA")
}

func isProductOwner(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var errorStruct ErrorJSON

		params := mux.Vars(r)
		productName := params["product"]

		var product Product
		err := db.Where("codename = ?", productName).First(&product).Error

		if err != nil {
			errorStruct.Message = "product not found"
			w.WriteHeader(http.StatusBadRequest)
			JSONResponse(errorStruct, w)
			return
		}

		email := GetClaim("email", r)

		var shop Shop
		db.Preload(clause.Associations).Take(&shop, "id = ?", product.ShopID)

		if shop.User.Email != email {
			errorStruct.Message = "you cannot modify this product"
			w.WriteHeader(http.StatusUnauthorized)
			JSONResponse(errorStruct, w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isLocationOwner(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var errorStruct ErrorJSON

		params := mux.Vars(r)
		locationName := params["location"]
		shopName := params["shop"]

		var location Location
		err := db.Where("name = ?", locationName).First(&location).Error

		if err != nil {
			errorStruct.Message = "location not found"
			w.WriteHeader(http.StatusBadRequest)
			JSONResponse(errorStruct, w)
			return
		}

		email := GetClaim("email", r)

		var shop Shop
		db.Preload(clause.Associations).Where("shopname = ?", shopName).Take(&shop)

		if shop.User.Email != email {
			errorStruct.Message = "you cannot modify this product"
			w.WriteHeader(http.StatusUnauthorized)
			JSONResponse(errorStruct, w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isAuthorized(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var errorStruct ErrorJSON

		accessTokenCookie, err := r.Cookie("Access-Token")
		if err == nil {
			claims := jwt.MapClaims{}
			token, err := jwt.ParseWithClaims(accessTokenCookie.Value, claims, func(token *jwt.Token) (interface{}, error) {
				_, ok := token.Method.(*jwt.SigningMethodHMAC)
				if !ok {
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

			if token.Valid {
				ctx := context.WithValue(r.Context(), ctxKey{}, claims)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		errorStruct.Message = "unauthorized"
		JSONResponse(errorStruct, w)
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
