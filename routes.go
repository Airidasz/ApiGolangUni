package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
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
	r.Handle("/shops", isAuthorized(CreateShopHandler)).Methods("POST")
	r.Handle("/shop/{shop}", isAuthorized(UpdateShopHandler)).Methods("PUT")
	r.Handle("/shop/{shop}", isAuthorized(DeleteShopHandler)).Methods("DELETE")

	// ========================== Locations ==============================
	r.HandleFunc("/shop/{shopid}/locations", GetLocationsHandler).Methods("GET")
	r.HandleFunc("/shop/{shopid}/location/{locationid}", GetLocationHandler).Methods("GET")
	r.Handle("/shop/{shopid}/locations", isAuthorized(CreateLocationHandler)).Methods("POST")
	r.Handle("/shop/{shopid}/locations", isAuthorized(DeleteLocationsHandler)).Methods("DELETE")
	r.Handle("/shop/{shopid}/location/{locationid}", isAuthorized(UpdateLocationHandler)).Methods("PUT")
	r.Handle("/shop/{shopid}/location/{locationid}", isAuthorized(DeleteLocationHandler)).Methods("DELETE")

	// ========================== Products ==============================
	r.HandleFunc("/products", GetProductsHandler).Methods("GET")
	r.HandleFunc("/product/{productid}", GetProductHandler).Methods("GET")
	r.Handle("/products", isAuthorized(CreateProductHandler)).Methods("POST")
	r.Handle("/product/{productid}", isAuthorized(UpdateProductHandler)).Methods("PUT")
	r.Handle("/product/{productid}", isAuthorized(DeleteProductHandler)).Methods("DELETE")

	// ========================== Categories ==============================
	r.HandleFunc("/categories", GetCategoriesHandler).Methods("GET")
	r.HandleFunc("/category/{categoryid}", GetCategoryHandler).Methods("GET")
	r.Handle("/categories", isAdmin(CreateCategoryHandler)).Methods("POST")
	r.Handle("/category/{categoryid}", isAdmin(UpdateCategoryHandler)).Methods("PUT")
	r.Handle("/category/{categoryid}", isAdmin(DeleteCategoryHandler)).Methods("DELETE")

	fmt.Println("Opened a server on port :8080")

	c := cors.New(cors.Options{
		AllowedOrigins:     []string{"http://localhost:3000", os.Getenv("API_URL"), "http://localhost"},
		AllowCredentials:   true,
		AllowedMethods:     []string{"GET", "HEAD", "POST", "PUT", "OPTIONS", "DELETE"},
		OptionsPassthrough: true,
		ExposedHeaders:     []string{"Set-Cookie"},
	})

	handler := c.Handler(r)

	http.ListenAndServe(":8080", handler)
}
