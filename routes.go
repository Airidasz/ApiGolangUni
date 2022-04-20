package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func HandleRequests() {
	r := mux.NewRouter()

	r.PathPrefix("/images/").Handler(http.FileServer(http.Dir(".")))

	r.HandleFunc("/", LandingPage)
	// ========================== Auth ==============================
	r.HandleFunc("/login", Login).Methods("POST")
	r.HandleFunc("/logout", Logout).Methods("POST")
	r.HandleFunc("/register", CreateAccount).Methods("POST")
	r.HandleFunc("/checkmail", CheckEmail).Methods("POST")
	r.HandleFunc("/refresh", RefreshTokens).Methods("POST")

	// ========================== Shops ==============================
	r.HandleFunc("/shops", GetShops).Methods("GET")
	r.HandleFunc("/shop/orders", isAuthorized(GetShopOrders)).Methods("GET")
	r.HandleFunc("/shop/{shop}", GetShop).Methods("GET")
	r.HandleFunc("/shops", isAuthorized(CreateShop)).Methods("POST")
	r.HandleFunc("/shop", isAuthorized(UpdateShop)).Methods("PUT")
	// r.HandleFunc("/shop/{shop}", isAuthorized(isShopOwner(DeleteShop))).Methods("DELETE")

	// ========================== Products ==============================
	r.HandleFunc("/products", WithContext(GetProducts)).Methods("GET")
	r.HandleFunc("/product/{product}", WithContext(GetProduct)).Methods("GET")
	r.HandleFunc("/products", isAuthorized(AddEditProduct)).Methods("POST")
	r.HandleFunc("/product/{product}", isAuthorized(isProductOwner(AddEditProduct))).Methods("PUT")
	r.HandleFunc("/product/{product}", isAuthorized(isProductOwner(DeleteProduct))).Methods("DELETE")

	// ========================== Categories ==============================
	r.HandleFunc("/categories", GetCategories).Methods("GET")
	r.HandleFunc("/category/{categoryid}", GetCategory).Methods("GET")
	r.HandleFunc("/categories", isAdmin(CreateCategory)).Methods("POST")
	r.HandleFunc("/category/{categoryid}", isAdmin(UpdateCategory)).Methods("PUT")
	r.HandleFunc("/category/{categoryid}", isAdmin(DeleteCategory)).Methods("DELETE")

	// ========================== Orders ==============================
	r.HandleFunc("/orders", PlaceOrder).Methods("POST")
	r.HandleFunc("/orders", isAdmin(ChangeOrder)).Methods("PUT")
	r.HandleFunc("/orders", isAuthorized(GetOrders)).Methods("GET")

	// CORS policy
	credentials := handlers.AllowCredentials()
	methods := handlers.AllowedMethods([]string{"POST", "GET", "PUT", "DELETE"})

	corsUrls := strings.Split(os.Getenv("CORS_ALLOWED_URLS"), ",")
	origins := handlers.AllowedOrigins(corsUrls)

	fmt.Println("Opened a server on port :8080")
	http.ListenAndServe(":8080", handlers.CORS(credentials, methods, origins)(r))
}
