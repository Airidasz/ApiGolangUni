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

func CreateTempProduct(name string, shop_codename string) Product {
	var shop Shop
	db.Take(&shop, "codename = ?", shop_codename)

	tempProduct := Product{
		Name:     &name,
		Codename: name,
		Price:    decimal.NewFromInt(10000),
		Quantity: 10000,
		ShopID:   shop.ID,
		Public:   true,
	}

	db.Save(&tempProduct)
	return tempProduct
}

func (app *app) CloseDbTest() {
	db, _ := app.DB.DB()
	db.Close()
}

func TestGetProduct(t *testing.T) {
	app := NewApp().InitRouter().InitDB(".env-test")

	tempProduct := CreateTempProduct("getProductTest", "seller_shop")

	t.Cleanup(func() {
		app.DB.Unscoped().Delete(&Product{}, "name = ?", "getProductTest")

		app.CloseDbTest()
	})

	cases := []TestStruct{
		{
			name:     "GetCorrect",
			body:     map[string]interface{}{"codename": tempProduct.Codename},
			response: jsonpath.Chain().Equal("name", *tempProduct.Name).Equal("price", ToString(tempProduct.Price)).Equal("quantity", float64(tempProduct.Quantity)),
			expected: http.StatusOK,
		},
		{
			name:     "DoesNotExist",
			body:     map[string]interface{}{"codename": "doesNotExist"},
			expected: http.StatusNotFound,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			url := fmt.Sprintf("/product/%s", c.body["codename"])

			test := apitest.New(c.name).
				Handler(app.Router).
				Get(url)

			response := test.Expect(t).Status(c.expected)

			if c.response != nil {
				response.Assert(c.response.End())
			}

			response.End()
		})
	}

}

func TestEditProduct(t *testing.T) {
	app := NewApp().InitRouter().InitDB(".env-test")

	_, SellerOneToken, _ := InitAccount(app, "seller")
	_, SellerTwoToken, _ := InitAccount(app, "seller2")

	tempProduct := CreateTempProduct("testEditProduct1", "seller_shop")

	t.Cleanup(func() {
		app.DB.Unscoped().Delete(&Product{}, "name ~ ?", "testEditProduct")
		app.CloseDbTest()
	})

	cases := []TestStruct{
		{
			name:     "UnauthorizedNoToken",
			body:     map[string]interface{}{"name": "testEditProduct11", "price": 40, "public": false, "quantity": -10},
			expected: http.StatusUnauthorized,
		},
		{
			name:        "UnauthorizedNotOwner",
			body:        map[string]interface{}{"name": "testEditProduct11", "price": 40, "public": false, "quantity": -10},
			accessToken: &SellerTwoToken,
			expected:    http.StatusUnauthorized,
		},
		{
			name:        "QuantityCannotBeBelowZero",
			body:        map[string]interface{}{"name": "testEditProduct11", "price": 40, "public": false, "quantity": -10},
			accessToken: &SellerOneToken,
			expected:    http.StatusBadRequest,
		},
		{
			name:        "PriceCannotBeBelowZeor",
			body:        map[string]interface{}{"name": "testEditProduct22", "price": -40, "public": false, "quantity": 10},
			accessToken: &SellerOneToken,
			expected:    http.StatusBadRequest,
		},
		{
			name:        "EditCorrect",
			body:        map[string]interface{}{"name": "testEditProduct33", "price": 40, "public": false, "quantity": 10},
			accessToken: &SellerOneToken,
			response:    jsonpath.Chain().Equal("name", "testEditProduct33").Equal("price", "40"),
			expected:    http.StatusOK,
		},
	}

	url := fmt.Sprintf("/product/%s", tempProduct.Codename)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			test := apitest.New(c.name).
				Handler(app.Router).
				Put(url)

			if c.accessToken != nil {
				test.Cookie("Access-Token", *c.accessToken)
			}

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
	app := NewApp().InitRouter().InitDB(".env-test")

	_, accessToken, _ := InitAccount(app, "seller")

	t.Cleanup(func() {
		app.DB.Unscoped().Delete(&Product{}, "name ~ ?", "testAddProduct")
		app.CloseDbTest()
	})

	cases := []TestStruct{
		{
			name:     "UnauthorizedNoShop",
			body:     map[string]interface{}{"price": 40, "public": false, "quantity": 10},
			expected: http.StatusUnauthorized,
		},
		{
			name:        "NameCannotBeEmpty",
			body:        map[string]interface{}{"price": 40, "public": false, "quantity": 10},
			accessToken: &accessToken,
			expected:    http.StatusBadRequest,
		},
		{
			name:        "QuantityCannotBeBelowZero",
			body:        map[string]interface{}{"name": "testAddProduct1", "price": 40, "public": false, "quantity": -10},
			accessToken: &accessToken,
			expected:    http.StatusBadRequest,
		},
		{
			name:        "PriceCannotBeBelowZeor",
			body:        map[string]interface{}{"name": "testAddProduct2", "price": -40, "public": false, "quantity": 10},
			accessToken: &accessToken,
			expected:    http.StatusBadRequest,
		},
		{
			name:        "ProductCreateSuccess",
			body:        map[string]interface{}{"name": "testAddProduct3", "price": 40, "public": false, "quantity": 10},
			accessToken: &accessToken,
			expected:    http.StatusCreated,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			test := apitest.New(c.name).
				Handler(app.Router).
				Post("/products")

			if c.accessToken != nil {
				test.Cookie("Access-Token", *c.accessToken)
			}

			for key, value := range c.body {
				test.FormData(key, fmt.Sprint(value))
			}

			test.Expect(t).Status(c.expected).End()
		})
	}
}
