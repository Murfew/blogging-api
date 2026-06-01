package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Post struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Category  string    `json:"category"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Application struct {
	pool *pgxpool.Pool
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ok")
}

func (app *Application) handleCreatePost(w http.ResponseWriter, r *http.Request) {
	var post Post
	if err := json.NewDecoder(r.Body).Decode(&post); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if post.Title == "" || post.Content == "" || post.Category == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "title, content and category are required"})
		return
	}

	query := "INSERT INTO posts (title, content, category, tags) VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at"
	err := app.pool.QueryRow(context.Background(), query, post.Title, post.Content, post.Category, post.Tags).Scan(&post.ID, &post.CreatedAt, &post.UpdatedAt)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("failed to add post: %v", err)})
		return
	}

	writeJSON(w, http.StatusCreated, post)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func createPool() *pgxpool.Pool {
	context := context.Background()
	connStr := os.Getenv("DATABASE_URL")

	pool, err := pgxpool.New(context, connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create connection pool: %v\n", err)
		os.Exit(1)
	}

	if err := pool.Ping(context); err != nil {
		fmt.Fprintf(os.Stderr, "Ping failed: %v\n", err)
		os.Exit(1)
	}

	return pool
}

func main() {
	app := &Application{pool: createPool()}
	defer app.pool.Close()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("POST /posts", app.handleCreatePost)

	fmt.Println("Server is running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))

}
