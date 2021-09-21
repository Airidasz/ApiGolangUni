package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

// var signKey = []byte("key")

// func GenerateToken() (string, error) {
// 	token := jwt.New(jwt.SigningMethodHS256)

// 	claims := token.Claims.(jwt.MapClaims)

// 	claims["user"] = true

// 	tokenString, err := token.SignedString(signKey)
// 	if err != nil {
// 		fmt.Errorf("Something Went Wrong: %s", err.Error())
// 		return "", err
// 	}
// 	return tokenString, nil
// }

func LandingPage(w http.ResponseWriter, r *http.Request) {
	b := struct {
		Make    string `json:"make"`
		Model   string `json:"model"`
		Mileage int    `json:"mileage"`
	}{"Ford", "Taurus", 2000010}

	JSONResponse(b, w)
	//w.WriteHeader(http.StatusOK)
}

func Routes() {
	r := mux.NewRouter()
	r.HandleFunc("/", LandingPage)

	r.HandleFunc("/login", LandingPage).Methods("POST")
	r.HandleFunc("/register", LandingPage).Methods("POST")
	r.HandleFunc("/token", LandingPage).Methods("POST")

	//r.Handle("/shop", isAuthorized(LandingPage)).Methods("GET")
	r.HandleFunc("/shop", LandingPage).Methods("GET")
	r.HandleFunc("/shop", LandingPage).Methods("POST")
	r.HandleFunc("/shop/{id}", LandingPage).Methods("GET")
	r.HandleFunc("/shop/{id}", LandingPage).Methods("PATCH")
	r.HandleFunc("/shop/{id}", LandingPage).Methods("DELETE")

	r.HandleFunc("/shop/{id}/locations", LandingPage).Methods("GET")
	r.HandleFunc("/shop/{id}/locations", LandingPage).Methods("POST")
	r.HandleFunc("/shop/{id}/locations/{id}", LandingPage).Methods("GET")
	r.HandleFunc("/shop/{id}/locations/{id}", LandingPage).Methods("PATCH")
	r.HandleFunc("/shop/{id}/locations/{id}", LandingPage).Methods("DELETE")

	r.HandleFunc("/shop/{id}/products", LandingPage).Methods("GET")
	r.HandleFunc("/shop/{id}/products", LandingPage).Methods("POST")
	r.HandleFunc("/shop/{id}/products/{id}", LandingPage).Methods("GET")
	r.HandleFunc("/shop/{id}/products/{id}", LandingPage).Methods("PATCH")
	r.HandleFunc("/shop/{id}/products/{id}", LandingPage).Methods("DELETE")

	http.ListenAndServe(":80", r)
}
