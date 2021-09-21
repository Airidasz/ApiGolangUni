package main

import (
	"encoding/json"
	"net/http"
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

// var db *sql.DB

func main() {
	// // Capture connection properties.
	// cfg := mysql.Config{
	// 	User:   "root",
	// 	Passwd: "password",
	// 	Net:    "tcp",
	// 	Addr:   "127.0.0.1:3306",
	// 	DBName: "testest",
	// }
	// // Get a database handle.
	// var err error
	// db, err = sql.Open("mysql", cfg.FormatDSN())
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// pingErr := db.Ping()
	// if pingErr != nil {
	// 	log.Fatal(pingErr)
	// }
	// fmt.Println("Connected!")

	// var (
	// 	a      string
	// 	artist string
	// )
	// rows, err := db.Query("select * from album")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer rows.Close()
	// for rows.Next() {
	// 	err := rows.Scan(&a, &artist)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	log.Println(a, artist)
	// }
	// err = rows.Err()
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// defer rows.Close()

	Routes()
}
