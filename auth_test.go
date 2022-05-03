package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/steinfletcher/apitest"
	jsonpath "github.com/steinfletcher/apitest-jsonpath"
)

type TestStruct struct {
	name        string
	body        map[string]interface{}
	accessToken *string
	response    *jsonpath.AssertionChain
	success     bool
	expected    int
}

func TestRegister(t *testing.T) {
	cases := []TestStruct{
		{
			name:     "PasswordsDontMatch",
			body:     map[string]interface{}{"name": "testUser1", "email": "testUser1@email.com", "password": "password123", "repeatPassword": "123password"},
			expected: http.StatusBadRequest,
		},
		{
			name:     "InvalidEmail",
			body:     map[string]interface{}{"name": "testUser2", "email": "testUser2", "password": "password123", "repeatPassword": "password123"},
			expected: http.StatusBadRequest,
		},
		{
			name:     "RegistrationSuccessfull",
			body:     map[string]interface{}{"name": "testUser3", "email": "testUser3@email.com", "password": "password123", "repeatPassword": "password123"},
			success:  true,
			expected: http.StatusCreated,
		},
	}

	app := NewApp().InitRouter().InitDB(".env-test")

	t.Cleanup(func() {
		app.DB.Unscoped().Delete(&User{}, "name ~ ?", "testUser")
		app.CloseDbTest()
	})

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			body, _ := json.Marshal(c.body)

			test := apitest.New(c.name).
				Handler(app.Router).
				Post("/register").JSON(body).
				Expect(t).
				Status(c.expected)

			if c.success {
				test.CookiePresent("Access-Token")
				test.CookiePresent("Refresh-Token")
			} else {
				test.CookieNotPresent("Access-Token")
				test.CookieNotPresent("Refresh-Token")
			}

			test.End()
		})
	}
}

func TestLogin(t *testing.T) {
	cases := []TestStruct{
		{
			name:     "AccountDoesntExist",
			body:     map[string]interface{}{"name": "testLoginUser1", "password": "password123"},
			expected: http.StatusBadRequest,
		},
		{
			name:     "SpecialCharacters",
			body:     map[string]interface{}{"name": "aiūųęėųūčęųšūėįšūčęūųygčuyghuhiwihugwjiĖĮ", "password": "password123"},
			expected: http.StatusBadRequest,
		},
		{
			name:     "AccountExistsGetToken",
			body:     map[string]interface{}{"name": "admin", "password": "a"},
			success:  true,
			expected: http.StatusAccepted,
		},
	}

	app := NewApp().InitRouter().InitDB(".env-test")

	t.Cleanup(func() {
		app.CloseDbTest()
	})

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			body, _ := json.Marshal(c.body)

			test := apitest.New(c.name).
				Handler(app.Router).
				Post("/login").JSON(body).
				Expect(t).
				Status(c.expected)

			if c.success {
				test.CookiePresent("Access-Token")
				test.CookiePresent("Refresh-Token")
			} else {
				test.CookieNotPresent("Access-Token")
				test.CookieNotPresent("Refresh-Token")
			}

			test.End()
		})
	}
}
