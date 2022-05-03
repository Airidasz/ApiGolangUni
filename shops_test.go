package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/steinfletcher/apitest"
	jsonpath "github.com/steinfletcher/apitest-jsonpath"
)

func CreateTempShop(name string, user User) Shop {
	description := "testest"
	tempShop := Shop{
		Name:     &name,
		Codename: name,
		Address:  &description,
		UserID:   user.ID,
	}

	db.Save(&tempShop)
	return tempShop
}

func TestEditShop(t *testing.T) {
	app := NewApp().InitRouter().InitDB(".env-test")

	_, buyerToken, _ := InitAccount(app, "buyer")
	sellerOne, sellerToken, _ := InitAccount(app, "seller")

	t.Cleanup(func() {
		var shop Shop
		app.DB.Take(&shop, "user_id = ?", sellerOne.ID)
		name := "seller shop"
		codename := "seller_shop"
		shop.Name = &name
		shop.Codename = codename

		sellerOne.ShopCodename = &codename
		db.Save(&sellerOne)
		app.DB.Save(&shop)
		app.CloseDbTest()
	})

	cases := []TestStruct{
		{
			name:     "UnauthorizedNoToken",
			expected: http.StatusUnauthorized,
		},
		{
			name:        "UnauthorizedNotFarmer",
			accessToken: &buyerToken,
			expected:    http.StatusUnauthorized,
		},
		{
			name:        "SuccessEdit",
			body:        map[string]interface{}{"name": "testShopEdited2"},
			response:    jsonpath.Chain().Equal("name", "testShopEdited2"),
			accessToken: &sellerToken,
			expected:    http.StatusOK,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			test := apitest.New(c.name).
				Handler(app.Router).
				Put("/shop")

			if c.body != nil {
				body, _ := json.Marshal(c.body)
				fmt.Println(string(body[:]))
				test.JSON(body)
			}

			if c.accessToken != nil {
				test.Cookie("Access-Token", *c.accessToken)
			}

			response := test.Expect(t).Status(c.expected)

			if c.response != nil {
				response.Assert(c.response.End())
			}

			response.End()
		})
	}
}

func TestAddShop(t *testing.T) {
	app := NewApp().InitRouter().InitDB(".env-test")

	_, sellerOneToken, _ := InitAccount(app, "seller")
	_, buyerToken, _ := InitAccount(app, "buyer")
	sellerTwo, sellerTwoToken, _ := InitAccount(app, "seller2")

	t.Cleanup(func() {
		sellerTwo.ShopCodename = nil
		app.DB.Save(&sellerTwo)
		app.DB.Unscoped().Delete(&Shop{}, "name ~ ?", "testShop")

		app.CloseDbTest()
	})

	cases := []TestStruct{
		{
			name:     "UnauthorizedNoToken",
			expected: http.StatusUnauthorized,
		},
		{
			name:        "UnauthorizedNotFarmer",
			accessToken: &buyerToken,
			expected:    http.StatusUnauthorized,
		},
		{
			name:        "UnauthorizedAlreadyHaveShop",
			accessToken: &sellerOneToken,
			expected:    http.StatusBadRequest,
		},
		{
			name:        "BadBodyFormat",
			body:        map[string]interface{}{"asdasdasasd": "asdasdasdasd"},
			accessToken: &sellerTwoToken,
			expected:    http.StatusBadRequest,
		},
		{
			name:        "NameRequired",
			body:        map[string]interface{}{"address": "address12345678"},
			accessToken: &sellerTwoToken,
			expected:    http.StatusBadRequest,
		},
		{
			name:        "AddressRequired",
			body:        map[string]interface{}{"name": "testShop"},
			accessToken: &sellerTwoToken,
			expected:    http.StatusBadRequest,
		},
		{
			name:        "NameTaken",
			body:        map[string]interface{}{"address": "address12345678", "name": "seller shop"},
			accessToken: &sellerTwoToken,
			expected:    http.StatusConflict,
		},
		{
			name:        "SuccessAdd",
			body:        map[string]interface{}{"address": "address12345678", "name": "testShop"},
			response:    jsonpath.Chain().Equal("address", "address12345678").Equal("name", "testShop"),
			accessToken: &sellerTwoToken,
			expected:    http.StatusCreated,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			test := apitest.New(c.name).
				Handler(app.Router).
				Post("/shops")

			if c.body != nil {
				body, _ := json.Marshal(c.body)
				fmt.Println(string(body[:]))
				test.JSON(body)
			}

			if c.accessToken != nil {
				test.Cookie("Access-Token", *c.accessToken)
			}

			response := test.Expect(t).Status(c.expected)

			if c.response != nil {
				response.Assert(c.response.End())
			}

			response.End()
		})
	}
}
