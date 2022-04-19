package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

func LandingPage(w http.ResponseWriter, r *http.Request) {
	b := struct {
		Make    string `json:"make"`
		Model   string `json:"model"`
		Mileage int    `json:"mileage"`
	}{"Vienas", "Du", 1}

	JSONResponse(b, w)
}

func JSONResponse(response interface{}, w http.ResponseWriter) {
	json, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

func GetShopByEmail(email string, shop *Shop, preload bool, params ...string) (err error) {
	tx := db

	for _, param := range params {
		tx.Select(param)
	}

	if preload {
		tx.Preload(clause.Associations)
	}

	return db.Joins("left join users on shops.user_id = users.id").Where("users.email = ?", email).Take(shop).Error
}

func NameTaken(name string, model interface{}) (err error) {
	err = db.Take(model, "name = ?", name).Error
	if err == nil {
		return errors.New("name taken")
	}

	return nil
}

func GetClaim(name string, r *http.Request) *string {
	claims := r.Context().Value(ctxKey{})

	if claims == nil {
		return nil
	}

	jwtClaims := claims.(jwt.MapClaims)

	if val, ok := jwtClaims[name]; ok {
		claimString := fmt.Sprintf("%v", val)
		return &claimString
	}

	return nil
}

func GetSingleParameter(r *http.Request, key string) string {
	value := r.Form[key]

	if len(value) > 0 {
		return value[0]
	}

	return ""
}

func FileUpload(r *http.Request, formFile string, namePattern string) string {
	// FormFile returns the first file for the given key `myFile`
	// it also returns the FileHeader so we can get the Filename,
	// the Header and the size of the file
	file, _, err := r.FormFile("file")
	if err != nil {
		return ""
	}
	defer file.Close()

	// Create a temporary file within our temp-images directory that follows
	// a particular naming pattern
	tempFile, err := ioutil.TempFile("images", namePattern)
	if err != nil {
		return ""
	}
	defer tempFile.Close()

	// read all of the contents of our uploaded file into a
	// byte array
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return ""
	}
	// write this byte array to our temporary file
	tempFile.Write(fileBytes)

	return tempFile.Name()
}

func GenerateOrderIdentifier() string {
	generated, _ := uuid.NewRandom()
	return generated.String()[0:8]
}

func GenerateCodename(input string, hasSuffix bool) string {
	// regex for NOT word, whitespace or digit
	specialCharacters := regexp.MustCompile(`[^\w|\d|\s]`)

	codename := specialCharacters.ReplaceAllString(strings.ToLower(input), "")
	codename = strings.TrimSpace(codename)
	codename = strings.ReplaceAll(codename, " ", "-")

	// Replace Lithuanian letters
	for ltLetter, enLetter := range enLtLetterMap {
		codename = strings.ReplaceAll(codename, ltLetter, enLetter)
	}

	if !hasSuffix {
		return codename
	}

	generated, _ := uuid.NewRandom()
	return codename + "-" + generated.String()[0:4]
}

func Trim(s *string) *string {
	if s == nil {
		return nil
	}

	temp := strings.TrimSpace(*s)
	return &temp
}

func ParseCategories(data string) ([]Category, error) {
	var categories []Category

	var categoryIDs []string
	err := json.Unmarshal([]byte(data), &categoryIDs)

	if err != nil {
		return categories, errors.New("json parse error")
	}

	if len(categoryIDs) > 0 {
		db.Find(&categories, "id in ?", categoryIDs)
	}

	return categories, nil
}

func CreateTempUser(user User) error {
	if !emailRegex.MatchString(user.Email) {
		return errors.New("please enter a valid email address")
	}

	err := CheckEmailAvailability(user.Email)
	if err != nil {
		return err
	}

	return db.Create(&user).Error
}
