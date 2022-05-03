package main

import (
	"net/http"
	"testing"

	"github.com/steinfletcher/apitest"
)

func TestCourierGetDeliveries(t *testing.T) {
	app := NewApp().InitRouter().InitDB(".env-test")

	t.Cleanup(func() {
		app.CloseDbTest()
	})

	_, courierToken, _ := InitAccount(app, "courier")
	_, buyerToken, _ := InitAccount(app, "buyer")

	cases := []TestStruct{
		{
			name:     "UnauthorizedNoToken",
			expected: http.StatusUnauthorized,
		},
		{
			name:        "UnauthorizedNotCourier",
			accessToken: &buyerToken,
			expected:    http.StatusUnauthorized,
		},
		{
			name:        "Authorized",
			accessToken: &courierToken,
			expected:    http.StatusOK,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			test := apitest.New(c.name).
				Handler(app.Router).
				Get("/courier/deliveries")

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

func TestGetCouriers(t *testing.T) {
	app := NewApp().InitRouter().InitDB(".env-test")

	t.Cleanup(func() {
		app.CloseDbTest()
	})

	_, adminToken, _ := InitAccount(app, "admin")
	_, buyerToken, _ := InitAccount(app, "buyer")

	cases := []TestStruct{
		{
			name:     "UnauthorizedNoToken",
			expected: http.StatusUnauthorized,
		},
		{
			name:        "UnauthorizedNotAdmin",
			accessToken: &buyerToken,
			expected:    http.StatusUnauthorized,
		},
		{
			name:        "Authorized",
			accessToken: &adminToken,
			expected:    http.StatusOK,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			test := apitest.New(c.name).
				Handler(app.Router).
				Get("/couriers")

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
