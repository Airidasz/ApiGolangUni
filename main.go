package main

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func JSONResponse(response interface{}, w http.ResponseWriter) {
	json, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

var db *gorm.DB
var passwordRegex *regexp.Regexp
var emailRegex *regexp.Regexp
var signKey []byte

func main() {
	signKey = []byte("key")
	passwordRegex = regexp.MustCompile(`([A-Z].*=?)([0-9].*=?)|([0-9].*=?)([A-Z].*=?)`)
	emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

	var err error
	dsn := "root:password@tcp(127.0.0.1:3306)/testest?charset=utf8mb4&parseTime=True&loc=Local"

	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	db.AutoMigrate(&User{}, &Category{}, &Shop{}, &Location{}, &Product{}, &RefreshToken{})

	HandleRequests()
}
