package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golang-jwt/jwt"
)

func NameTaken(name string, model interface{}) (err error) {
	if db.Find(model, "name = ?", name).RowsAffected > 0 {
		return errors.New("name taken")
	}

	return nil
}

func GetClaim(name string, r *http.Request) string {
	claims := r.Context().Value(ctxKey{}).(jwt.MapClaims)
	return fmt.Sprintf("%v", claims[name])
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
