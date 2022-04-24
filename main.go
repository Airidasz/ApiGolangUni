package main

import (
	"fmt"
	"log"
	"os"
	"regexp"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB
var passwordRegex *regexp.Regexp
var emailRegex *regexp.Regexp
var signKey []byte

var enLtLetterMap = map[string]string{
	"ą": "a",
	"č": "c",
	"ę": "e",
	"ė": "e",
	"į": "i",
	"š": "s",
	"ų": "u",
	"ū": "u",
	"ž": "z",
}

func main() {
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	signKey = []byte(os.Getenv("JWT_SECRET"))
	passwordRegex = regexp.MustCompile(`([A-Z].*=?)([0-9].*=?)|([0-9].*=?)([A-Z].*=?)`)
	emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Europe/Vilnius", os.Getenv("DB_HOST"), os.Getenv("DB_USERNAME"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_PORT"))

	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})

	if err != nil {
		log.Fatal(err)
	}
	//make work?
	db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"")
	db.AutoMigrate(&User{}, &Category{}, &Shop{}, &Location{}, &Product{}, &RefreshToken{}, &OrderedProduct{}, &Order{})

	HandleRequests()
}
