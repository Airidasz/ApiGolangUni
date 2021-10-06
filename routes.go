package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func HandleRequests() {
	r := mux.NewRouter()
	r.HandleFunc("/", LandingPage)

	// ========================== Auth ==============================
	r.HandleFunc("/login", LoginHandler).Methods("POST")
	r.HandleFunc("/register", CreateAccountHandler).Methods("POST")
	r.HandleFunc("/refresh", RefreshTokenHandler).Methods("POST")

	// ========================== Shops ==============================
	r.HandleFunc("/shops", GetShopsHandler).Methods("GET")
	r.HandleFunc("/shop/{shopid}", GetShopHandler).Methods("GET")
	r.Handle("/shops", isAuthorized(CreateShopHandler)).Methods("POST")
	r.Handle("/shop/{shopid}", isAuthorized(isShopOwner(UpdateShopHandler))).Methods("PUT")
	r.Handle("/shop/{shopid}", isAuthorized(isShopOwner(DeleteShopHandler))).Methods("DELETE")

	// ========================== Locations ==============================
	r.HandleFunc("/shop/{shopid}/locations", GetLocationsHandler).Methods("GET")
	r.Handle("/shop/{shopid}/location/{locationid}", shopLocationValid(GetLocationHandler)).Methods("GET")
	r.Handle("/shop/{shopid}/locations", isAuthorized(isShopOwner(CreateLocationHandler))).Methods("POST")
	r.Handle("/shop/{shopid}/location/{locationid}", isAuthorized(isShopOwner(shopLocationValid(UpdateLocationHandler)))).Methods("PUT")
	r.Handle("/shop/{shopid}/location/{locationid}", isAuthorized(isShopOwner(shopLocationValid(DeleteLocationHandler)))).Methods("DELETE")

	// ========================== Products ==============================
	r.HandleFunc("/shop/{shopid}/products", GetProductsHandler).Methods("GET")
	r.Handle("/shop/{shopid}/product/{productid}", shopProductValid(GetProductHandler)).Methods("GET")
	r.Handle("/shop/{shopid}/products", isAuthorized(isShopOwner(CreateProductHandler))).Methods("POST")
	r.Handle("/shop/{shopid}/product/{productid}", isAuthorized(isShopOwner(shopProductValid(UpdateProductHandler)))).Methods("PUT")
	r.Handle("/shop/{shopid}/product/{productid}", isAuthorized(isShopOwner(shopProductValid(DeleteProductHandler)))).Methods("DELETE")

	// ========================== Categories ==============================
	r.HandleFunc("/categories", GetCategoriesHandler).Methods("GET")
	r.HandleFunc("/category/{categoryid}", GetCategoryHandler).Methods("GET")
	r.Handle("/categories", isAdmin(CreateCategoryHandler)).Methods("POST")
	r.Handle("/category/{categoryid}", isAdmin(UpdateCategoryHandler)).Methods("PUT")
	r.Handle("/category/{categoryid}", isAdmin(DeleteCategoryHandler)).Methods("DELETE")

	fmt.Println("Opened a server on port :8080")
	http.ListenAndServe(":8080", r)
}
