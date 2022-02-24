package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func HandleRequests() {
	r := mux.NewRouter()

	r.PathPrefix("/images/").Handler(http.FileServer(http.Dir(".")))

	r.HandleFunc("/", LandingPage)
	// ========================== Auth ==============================
	r.HandleFunc("/login", LoginHandler).Methods("POST")
	r.HandleFunc("/logout", LogoutHandler).Methods("POST")
	r.HandleFunc("/register", CreateAccountHandler).Methods("POST")
	r.HandleFunc("/refresh", RefreshTokenHandler).Methods("POST")

	// ========================== Shops ==============================
	r.HandleFunc("/shops", GetShopsHandler).Methods("GET")
	r.HandleFunc("/shop/{shop}", GetShopHandler).Methods("GET")
	r.HandleFunc("/shops", CreateShopHandler).Methods("POST")
	r.HandleFunc("/shop/{shop}", UpdateShopHandler).Methods("PUT")
	r.HandleFunc("/shop/{shop}", DeleteShopHandler).Methods("DELETE")

	// ========================== Products ==============================
	r.HandleFunc("/products", GetProductsHandler).Methods("GET")
	r.HandleFunc("/product/{product}", GetProductHandler).Methods("GET")
	r.HandleFunc("/products", CreateProductHandler).Methods("POST")
	r.HandleFunc("/product/{product}", UpdateProductHandler).Methods("PUT")
	r.HandleFunc("/product/{product}", DeleteProductHandler).Methods("DELETE")

	// ========================== Locations ==============================
	r.HandleFunc("/shop/{shopid}/locations", GetLocationsHandler).Methods("GET")
	r.HandleFunc("/shop/{shopid}/location/{locationid}", GetLocationHandler).Methods("GET")
	r.HandleFunc("/shop/{shopid}/locations", isAuthorized(CreateLocationHandler)).Methods("POST")
	r.HandleFunc("/shop/{shopid}/locations", isAuthorized(DeleteLocationsHandler)).Methods("DELETE")
	r.HandleFunc("/shop/{shopid}/location/{locationid}", isAuthorized(UpdateLocationHandler)).Methods("PUT")
	r.HandleFunc("/shop/{shopid}/location/{locationid}", isAuthorized(DeleteLocationHandler)).Methods("DELETE")

	// ========================== Categories ==============================
	r.HandleFunc("/categories", GetCategoriesHandler).Methods("GET")
	r.HandleFunc("/category/{categoryid}", GetCategoryHandler).Methods("GET")
	r.HandleFunc("/categories", isAdmin(CreateCategoryHandler)).Methods("POST")
	r.HandleFunc("/category/{categoryid}", isAdmin(UpdateCategoryHandler)).Methods("PUT")
	r.HandleFunc("/category/{categoryid}", isAdmin(DeleteCategoryHandler)).Methods("DELETE")

	fmt.Println("Opened a server on port :8080")

	http.ListenAndServe(":8080", r)
}
