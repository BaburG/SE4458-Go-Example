package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"text/template"
	"os"

	_ "github.com/lib/pq"
)

var tmpl = template.Must(template.ParseFiles("index.html"))

type Note struct {
	ID    int
	Title string
	Body  string
	CreatedAt string
}

// Connect to PostgreSQL with the specified database
func dbConnect(dbname string) (*sql.DB, error) {
    host := os.Getenv("DB_HOST")
    user := os.Getenv("DB_USER")
    password := os.Getenv("DB_PASSWORD")
    connStr := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable password=%s", host, user, dbname, password)
    return sql.Open("postgres", connStr)
}


// Initialize the database and table if they don’t exist
func initializeDB() {
    dbName := os.Getenv("DB_NAME")
    db, err := dbConnect(dbName)
    if err != nil {
        log.Fatal("Database connection error:", err)
    }
    defer db.Close()

    // Create the "notes" table if it doesn't exist
    query := `
    CREATE TABLE IF NOT EXISTS notes (
        id SERIAL PRIMARY KEY,
        title TEXT NOT NULL,
        body TEXT NOT NULL,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )`
    _, err = db.Exec(query)
    if err != nil {
        log.Fatal("Error creating table:", err)
    }
    fmt.Println("Database and table initialized successfully")
}

func getNotes(w http.ResponseWriter, r *http.Request) {
	db, err := dbConnect(os.Getenv("DB_NAME"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, title, body, created_at FROM notes ORDER BY created_at DESC")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	notes := []Note{}
	for rows.Next() {
		var note Note
		err := rows.Scan(&note.ID, &note.Title, &note.Body, &note.CreatedAt)
		if err != nil {
			log.Fatal(err)
		}
		notes = append(notes, note)
	}

	tmpl.Execute(w, notes)
}

func createNote(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		db, err := dbConnect(os.Getenv("DB_NAME"))
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		title := r.FormValue("title")
		body := r.FormValue("body")

		// Insert the note; PostgreSQL will automatically add the timestamp
		_, err = db.Exec("INSERT INTO notes (title, body) VALUES ($1, $2)", title, body)
		if err != nil {
			log.Fatal("Error inserting new note:", err)
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received %s request for %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	}
}

func main() {
	initializeDB() // Ensure database and table setup on startup
	http.HandleFunc("/", loggingMiddleware(getNotes))
	http.HandleFunc("/create", loggingMiddleware(createNote))
	http.Handle("/static/", loggingMiddleware(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))).ServeHTTP))
	fmt.Println("Starting server on :8080")
	log.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
