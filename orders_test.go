package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/steinfletcher/apitest"
)

func PlaceOrderTemp(t *testing.T) {
	OrderedProducts := []OrderedProduct{{Quantity: 1}}

	app := NewApp().InitRouter().InitDB(".env-test")

	t.Cleanup(func() {
		app.CloseDbTest()
	})

	buyer, _, _ := InitAccount(app, "buyer")

	cases := []TestStruct{
		{
			name:     "UnauthorizedNoToken",
			body:     map[string]interface{}{"address": "asd", "paymentType": 1, "user": buyer, "orderedProducts": OrderedProducts},
			expected: http.StatusUnauthorized,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			test := apitest.New(c.name).
				Handler(app.Router).
				Post("/orders")

			if c.accessToken != nil {
				test.Cookie("Access-Token", *c.accessToken)
			}

			if c.body != nil {
				body, _ := json.Marshal(c.body)
				test.JSON(body)
			}

			response := test.Expect(t).Status(c.expected)

			if c.response != nil {
				response.Assert(c.response.End())
			}

			response.End()
		})
	}
}
