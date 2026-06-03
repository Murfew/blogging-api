package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
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

func validateRequestBody(w http.ResponseWriter, r *http.Request, post *Post) bool {
	if err := json.NewDecoder(r.Body).Decode(&post); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return false
	}

	if post.Title == "" || post.Content == "" || post.Category == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "title, content and category are required"})
		return false
	}

	return true
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ok")
}

func (app *Application) handleCreatePost(w http.ResponseWriter, r *http.Request) {
	var post Post
	if !validateRequestBody(w, r, &post) {
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

func (app *Application) handleUpdatePost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("id must be of type int: %v", err)})
		return
	}

	var newPost Post
	if !validateRequestBody(w, r, &newPost) {
		return
	}

	query := "UPDATE posts SET title = $1, content = $2, category = $3, tags = $4, updated_at = $5 WHERE id = $6 RETURNING id, created_at, updated_at"
	err = app.pool.QueryRow(context.Background(), query, newPost.Title, newPost.Content, newPost.Category, newPost.Tags, time.Now(), id).Scan(&newPost.ID, &newPost.CreatedAt, &newPost.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: fmt.Sprintf("failed to find post with id %d: %v", id, err)})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("failed to update post (ID: %d): %v", id, err)})
		return
	}

	writeJSON(w, http.StatusOK, newPost)
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
	mux.HandleFunc("PUT /posts/{id}", app.handleUpdatePost)

	fmt.Println("Server is running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))

}
