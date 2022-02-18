package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/pbkdf2"
	"gorm.io/gorm/clause"
)

//CheckNameAvailability checks if a username is available
func CheckEmailAvailability(email string) error {
	//if no record of the email is found, returns an error
	if db.Find(&User{}, "email = ?", email).RowsAffected > 0 {
		return errors.New("this email is taken")
	}

	return nil
}

//CreateNewAccount creates an account if the sent data
//is correctly formatted
func PerformUserDataChecks(email string, password string, repeatedPassword string) (httpStatus int, err error) {
	if !emailRegex.MatchString(email) {
		return http.StatusNotAcceptable, errors.New("please enter a valid email")
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
	accessClaims := map[string]interface{}{
		"id":    user.ID,
		"email": user.Email,
		"admin": user.Admin,
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	accessToken, _ := GenerateToken(accessClaims)
	http.SetCookie(w, &http.Cookie{Name: "Access-Token", Value: accessToken, MaxAge: 60, SameSite: http.SameSiteNoneMode, Secure: true})

	refreshClaims := map[string]interface{}{
		"id":    user.ID,
		"email": user.Email,
		"admin": user.Admin,
		"exp":   time.Now().Add(time.Hour * 24 * 7).Unix(),
	}
	refreshToken, _ := GenerateToken(refreshClaims)

	refreshDatabaseEntry := RefreshToken{
		Token: refreshToken,
		Email: user.Email,
	}
	db.Create(&refreshDatabaseEntry)

	http.SetCookie(w, &http.Cookie{Name: "Refresh-Token", Value: refreshToken, HttpOnly: true, MaxAge: 60 * 60 * 24 * 7, SameSite: http.SameSiteNoneMode, Secure: true})

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
				isAdmin := claims["admin"].(bool)

				if isAdmin {
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

func shopProductValid(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		productID, err := strconv.Atoi(params["productid"])

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			JSONResponse(ErrorJSON{
				Message: "product not found",
			}, w)
			return
		}

		var product Product
		db.Take(&product, productID)

		if len(product.Name) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			JSONResponse(ErrorJSON{
				Message: "product not found",
			}, w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func shopLocationValid(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		shopID, err1 := strconv.Atoi(params["shopid"])
		locationID, err2 := strconv.Atoi(params["locationid"])

		if err1 != nil || err2 != nil {
			JSONResponse(ErrorJSON{
				Message: "bad shop id or location id",
			}, w)
			return
		}

		var location Location
		db.Take(&location, locationID)

		if location.ShopID != uint(shopID) {
			w.WriteHeader(http.StatusBadRequest)
			JSONResponse(ErrorJSON{
				Message: "this shop does not have this location",
			}, w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isShopOwner(shopID uint, r *http.Request) bool {
	// Get JWT claims from context
	claims := r.Context().Value(ctxKey{}).(jwt.MapClaims)
	email := fmt.Sprintf("%v", claims["email"])

	var shop Shop
	db.Preload(clause.Associations).Take(&shop, shopID)

	return shop.User.Email == email

}

func isProductOwner(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)

		// Get product id from url
		productID, err := strconv.Atoi(params["productid"])

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			JSONResponse(ErrorJSON{
				Message: "product not found",
			}, w)
			return
		}

		//Check if product exists in database
		var product Product
		db.Take(&product, productID)

		if len(product.Name) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			JSONResponse(ErrorJSON{
				Message: "product not found",
			}, w)
			return
		}

		// Check for ownership
		if !isShopOwner(product.ShopID, r) {
			w.WriteHeader(http.StatusUnauthorized)
			JSONResponse(ErrorJSON{
				Message: "unauthorized",
			}, w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isLocationOwner(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)

		// Get product id from url
		locationID, err := strconv.Atoi(params["locationid"])

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			JSONResponse(ErrorJSON{
				Message: "product not found",
			}, w)
			return
		}

		//Check if product exists in database
		var location Location
		db.Take(&location, locationID)

		if len(location.Type) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			JSONResponse(ErrorJSON{
				Message: "product not found",
			}, w)
			return
		}

		// Check for ownership
		if !isShopOwner(location.ShopID, r) {
			w.WriteHeader(http.StatusUnauthorized)
			JSONResponse(ErrorJSON{
				Message: "unauthorized",
			}, w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isAuthorized(next http.HandlerFunc) http.HandlerFunc {
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
				ctx := context.WithValue(r.Context(), ctxKey{}, claims)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		w.WriteHeader(http.StatusUnauthorized)
		JSONResponse(ErrorJSON{
			Message: "unauthorized",
		}, w)

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
