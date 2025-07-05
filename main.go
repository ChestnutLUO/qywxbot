package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB
var tmpl *template.Template

type Bot struct {
	ID  int
	URL string
}

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./bots.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	createTable()

	tmpl = template.Must(template.ParseFiles("templates/index.html"))

	http.HandleFunc("/", handler)
	log.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func createTable() {
	createTableSQL := `CREATE TABLE IF NOT EXISTS bots (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"url" TEXT
	  );`

	statement, err := db.Prepare(createTableSQL)
	if err != nil {
		log.Fatal(err)
	}
	statement.Exec()
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		url := r.FormValue("url")
		if url == "" {
			http.Error(w, "URL is required", http.StatusBadRequest)
			return
		}

		id, err := insertBot(url)
		if err != nil {
			http.Error(w, "Failed to register bot", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Bot registered with ID: %d", id)
		return
	}

	tmpl.Execute(w, nil)
}

func insertBot(url string) (int64, error) {
	insertSQL := "INSERT INTO bots(url) VALUES (?)"
	statement, err := db.Prepare(insertSQL)
	if err != nil {
		return 0, err
	}
	result, err := statement.Exec(url)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}
