package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/steinfletcher/apitest"
	jsonpath "github.com/steinfletcher/apitest-jsonpath"
)

func InitAccount(a *app, name string) (user User, accessToken string, refreshToken string) {
	var seller User
	a.DB.Take(&seller, "name = ?", name)

	w := httptest.NewRecorder()

	access, refresh := MakeTokens(w, seller)
	return seller, access, refresh
}

func (app * app) CleanupAfterTest() {
	app.DB.Unscoped().Delete(&Product{}, "name ~ ?", "test")
	app.DB.Unscoped().Delete(&Category{}, "name ~ ?", "test")
	app.DB.Unscoped().Delete(&RefreshToken{})

	db, _ := app.DB.DB()
	db.Close()
}

func TestEditProduct(t *testing.T) {
	app := NewApp().InitRouter().InitDB(".env-test")

	_, accessToken, _ := InitAccount(app, "seller")

	var shop Shop
	db.Take(&shop, "codename = ?", "seller_shop")

	name := "testEditProduct1"
	tempProduct := Product{
		Name:     &name,
		Codename: "testEditProduct1",
		Price:    decimal.NewFromInt(10000),
		Quantity: 10000,
		ShopID:   shop.ID,
	}

	db.Save(&tempProduct)

	t.Cleanup(func() {
		app.CleanupAfterTest()
	})

	cases := []TestStruct{
		{
			name:     "QuantityCannotBeBelowZero",
			body:     map[string]interface{}{"name": "testEditProduct11", "price": 40, "public": false, "quantity": -10},
			expected: http.StatusBadRequest,
		},
		{
			name:     "PriceCannotBeBelowZeor",
			body:     map[string]interface{}{"name": "testEditProduct22", "price": -40, "public": false, "quantity": 10},
			expected: http.StatusBadRequest,
		},
		{
			name:     "EditCorrect",
			body:     map[string]interface{}{"name": "testEditProduct33", "price": 40, "public": false, "quantity": 10},
			response: jsonpath.Chain().Equal("name", "testEditProduct33").Equal("price", "40").NotEqual("quantity", "10000"),
			expected: http.StatusOK,
		},
	}

	url := fmt.Sprintf("/product/%s", tempProduct.Codename)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			test := apitest.New(c.name).
				Handler(app.Router).
				Put(url).Cookie("Access-Token", accessToken)

			for key, value := range c.body {
				test.FormData(key, fmt.Sprint(value))
			}

			response := test.Expect(t).Status(c.expected)

			if c.response != nil {
				response.Assert(c.response.End())
			}

			response.End()
		})
	}
}

func TestAddProduct(t *testing.T) {
	// Name        *string   `json:"name"`
	// Description *string   `json:"description"`
	// Categories  *[]string `json:"categories"`
	// Price       *float64  `json:"amount"`
	// Public      *bool     `json:"public"`
	// Quantity    *int      `json:"quantity"`
	cases := []TestStruct{
		{
			name:     "NameCannotBeEmpty",
			body:     map[string]interface{}{"price": 40, "public": false, "quantity": 10},
			expected: http.StatusBadRequest,
		},
		{
			name:     "QuantityCannotBeBelowZero",
			body:     map[string]interface{}{"name": "testProduct1", "price": 40, "public": false, "quantity": -10},
			expected: http.StatusBadRequest,
		},
		{
			name:     "PriceCannotBeBelowZeor",
			body:     map[string]interface{}{"name": "testProduct2", "price": -40, "public": false, "quantity": 10},
			expected: http.StatusBadRequest,
		},
		{
			name:     "ProductCreateSuccess",
			body:     map[string]interface{}{"name": "testProduct3", "price": 40, "public": false, "quantity": 10},
			expected: http.StatusCreated,
		},
	}

	app := NewApp().InitRouter().InitDB(".env-test")

	_, accessToken, _ := InitAccount(app, "seller")

	t.Cleanup(func() {
		app.CleanupAfterTest()
	})

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			test := apitest.New(c.name).
				Handler(app.Router).
				Post("/products").Cookie("Access-Token", accessToken)

			for key, value := range c.body {
				test.FormData(key, fmt.Sprint(value))
			}

			test.Expect(t).Status(c.expected).End()
		})
	}
}
