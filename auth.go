package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"golang.org/x/crypto/pbkdf2"
)

func MakeTokens(w http.ResponseWriter, email string) {
	accessClaims := map[string]interface{}{
		"email": email,
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	accessToken, _ := GenerateToken(accessClaims)
	http.SetCookie(w, &http.Cookie{Name: "Access-Token", Value: accessToken})

	refreshClaims := map[string]interface{}{
		"email": email,
		"exp":   time.Now().Add(time.Hour * 24 * 7).Unix(),
	}
	refreshToken, _ := GenerateToken(refreshClaims)

	refreshDatabaseEntry := RefreshToken{
		Token: refreshToken,
		Email: email,
	}
	db.Create(&refreshDatabaseEntry)

	http.SetCookie(w, &http.Cookie{Name: "Refresh-Token", Value: refreshToken})
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

func isAuthorized(endpoint func(http.ResponseWriter, *http.Request)) http.Handler {
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
				fmt.Fprint(w, err.Error())
				return
			}

			if token.Valid {
				ctx := context.WithValue(r.Context(), ctxKey{}, claims["email"])
				endpoint(w, r.WithContext(ctx))
			}
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, "Not Authorized")
		}
	})
}

func GenerateToken(claimsMap map[string]interface{}) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)
	for k, e := range claimsMap {
		claims[k] = e
	}

	tokenString, err := token.SignedString(signKey)

	if err != nil {
		return "", err
	}

	return tokenString, nil
}
