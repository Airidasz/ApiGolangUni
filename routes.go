package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func (a *app) InitRouter() *app {
	r := mux.NewRouter()

	r.PathPrefix("/images/").Handler(http.FileServer(http.Dir(".")))

	r.HandleFunc("/", LandingPage)
	// ========================== Auth ==============================
	r.HandleFunc("/login", Login).Methods("POST")            // Tested
	r.HandleFunc("/logout", Logout).Methods("POST")          // -
	r.HandleFunc("/register", CreateAccount).Methods("POST") // Tested
	r.HandleFunc("/checkmail", CheckEmail).Methods("POST")   // -
	r.HandleFunc("/refresh", RefreshTokens).Methods("POST")  // -

	// ========================== Shops ==============================
	r.HandleFunc("/shops", GetShops).Methods("GET")                           // -
	r.HandleFunc("/shop/orders", isFarmer(GetShopOrders)).Methods("GET")      // ?
	r.HandleFunc("/shop/orders/{id}", isFarmer(EditShopOrder)).Methods("PUT") // ?
	r.HandleFunc("/shop/{shop}", GetShop).Methods("GET")                      // ?
	r.HandleFunc("/shops", isFarmer(CreateShop)).Methods("POST")              // Tested
	r.HandleFunc("/shop", isFarmer(UpdateShop)).Methods("PUT")                // Tested

	// ========================== Products ==============================
	r.HandleFunc("/products", WithContext(GetProducts)).Methods("GET")                                // -
	r.HandleFunc("/product/{product}", WithContext(GetProduct)).Methods("GET")                        // Tested
	r.HandleFunc("/products", isAuthorized(AddEditProduct)).Methods("POST")                           // Tested
	r.HandleFunc("/product/{product}", isAuthorized(isProductOwner(AddEditProduct))).Methods("PUT")   // Tested
	r.HandleFunc("/product/{product}", isAuthorized(isProductOwner(DeleteProduct))).Methods("DELETE") // ?

	// ========================== Categories ==============================
	r.HandleFunc("/categories", GetCategories).Methods("GET")                         // - know admin middleware works
	r.HandleFunc("/category/{categoryid}", GetCategory).Methods("GET")                // -
	r.HandleFunc("/categories", isAdmin(CreateCategory)).Methods("POST")              // -
	r.HandleFunc("/category/{categoryid}", isAdmin(UpdateCategory)).Methods("PUT")    // -
	r.HandleFunc("/category/{categoryid}", isAdmin(DeleteCategory)).Methods("DELETE") // -

	// ========================== Orders ==============================
	r.HandleFunc("/orders", PlaceOrder).Methods("POST")                                    // TBD BUTINA
	r.HandleFunc("/orders/{ordernumber}", isAdmin(ChangeOrder)).Methods("PUT")             // TBD
	r.HandleFunc("/orders/{ordernumber}/cancel", isAuthorized(CancelOrder)).Methods("PUT") // TBD
	r.HandleFunc("/orders", isAuthorized(GetOrders)).Methods("GET")                        // -

	// ========================== Couriers ==============================
	r.HandleFunc("/couriers", isAdmin(GetCouriers)).Methods("GET")               // Tested
	r.HandleFunc("/courier/deliveries", isCourier(GetDeliveries)).Methods("GET") // Tested
	r.HandleFunc("/courier/pickups", isCourier(GetPickups)).Methods("GET")       // - same as deliveries

	a.Router = r
	return a
}

func (a *app) Start() {
	// CORS policy
	credentials := handlers.AllowCredentials()
	methods := handlers.AllowedMethods([]string{"POST", "GET", "PUT", "DELETE"})

	corsUrls := strings.Split(os.Getenv("CORS_ALLOWED_URLS"), ",")
	origins := handlers.AllowedOrigins(corsUrls)

	fmt.Println("Opened a server on port :8080")
	http.ListenAndServe(":8080", handlers.CORS(credentials, methods, origins)(a.Router))
}
