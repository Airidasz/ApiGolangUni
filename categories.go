package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

// ========================== Categories ==============================
func GetCategories(w http.ResponseWriter, r *http.Request) {
	var categories []Category
	db.Order("created_at desc").Find(&categories)
	JSONResponse(categories, w)
}

func GetCategory(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	caregoryID := params["categoryid"]

	var category Category

	err := db.Take(&category, caregoryID).Error
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	JSONResponse(&category, w)
}

func CreateCategory(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)

	var request Category

	request.File = FileUpload(r, "file", "category-*.png")
	name := r.FormValue("name")

	if len(name) == 0 {
		Response(w, http.StatusBadRequest, "vardas yra privalomas")
		return
	}

	request.Name = &name
	request.Codename = GenerateCodename(name, false)
	db.Create(&request)

	w.WriteHeader(http.StatusCreated)
}

func UpdateCategory(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	categoryID := params["categoryid"]

	r.ParseMultipartForm(10 << 20)

	var category Category

	err := db.Take(&category, "id = ?", categoryID).Error
	if err != nil {
		Response(w, http.StatusBadRequest, "kategorijos nepavyko rasti")
		return
	}

	image := FileUpload(r, "file", "category-*.png")
	name := r.FormValue("name")

	if len(name) > 0 && *category.Name != name {
		*category.Name = name
		category.Codename = GenerateCodename(name, false)
	}

	if len(image) > 0 {
		category.File = image
	}

	db.Save(&category)
}

func DeleteCategory(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	caregoryID := params["categoryid"]

	db.Unscoped().Delete(&Category{}, "id = ?", caregoryID)
}

// ===================================================================
