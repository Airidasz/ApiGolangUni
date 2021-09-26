package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

func HandleRequests() {
	r := mux.NewRouter()
	r.HandleFunc("/", LandingPage)

	r.HandleFunc("/login", LoginHandler).Methods("POST")
	r.HandleFunc("/register", CreateAccountHandler).Methods("POST")
	r.HandleFunc("/refresh", RefreshTokenHandler).Methods("POST")

	r.HandleFunc("/shop", GetShopsHandler).Methods("GET")
	r.HandleFunc("/shop/{shopid}", GetShopHandler).Methods("GET")
	r.Handle("/shop", isAuthorized(CreateShopHandler)).Methods("POST")
	r.Handle("/shop/{shopid}", isAuthorized(LandingPage)).Methods("PUT")
	r.Handle("/shop/{shopid}", isAuthorized(LandingPage)).Methods("DELETE")

	r.HandleFunc("/shop/{shopid}/locations", GetLocationsHandler).Methods("GET")
	r.HandleFunc("/shop/{shopid}/locations/{locationsid}", LandingPage).Methods("GET")
	r.Handle("/shop/{shopid}/locations", isAuthorized(LandingPage)).Methods("POST")
	r.Handle("/shop/{shopid}/locations/{locationsid}", isAuthorized(LandingPage)).Methods("PUT")
	r.Handle("/shop/{shopid}/locations/{locationsid}", isAuthorized(LandingPage)).Methods("DELETE")

	r.HandleFunc("/shop/{shopid}/products", GetProductsHandler).Methods("GET")
	r.HandleFunc("/shop/{shopid}/products/{productsid}", LandingPage).Methods("GET")
	r.Handle("/shop/{shopid}/products", isAuthorized(LandingPage)).Methods("POST")
	r.Handle("/shop/{shopid}/products/{productsid}", isAuthorized(LandingPage)).Methods("PUT")
	r.Handle("/shop/{shopid}/products/{productsid}", isAuthorized(LandingPage)).Methods("DELETE")

	r.HandleFunc("/category", LandingPage).Methods("GET")
	r.HandleFunc("/category/{categoryid}", LandingPage).Methods("GET")
	r.Handle("/category", isAuthorized(LandingPage)).Methods("POST")
	r.Handle("/category/{categoryid}", isAuthorized(LandingPage)).Methods("PUT")
	r.Handle("/category/{categoryid}", isAuthorized(LandingPage)).Methods("DELETE")

	http.ListenAndServe(":80", r)
}
