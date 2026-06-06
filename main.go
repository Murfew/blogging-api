package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Post struct {
	ID        int       `json:"id"         db:"id"`
	Title     string    `json:"title"      db:"title"`
	Content   string    `json:"content"    db:"content"`
	Category  string    `json:"category"   db:"category"`
	Tags      []string  `json:"tags"       db:"tags"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type Application struct {
	pool *pgxpool.Pool
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func validateRequestBody(w http.ResponseWriter, r *http.Request, body *Post) bool {
	if err := json.NewDecoder(r.Body).Decode(body); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return false
	}

	if body.Title == "" || body.Content == "" || body.Category == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "title, content and category are required"})
		return false
	}

	return true
}

func validateIDPathParam(w http.ResponseWriter, r *http.Request, id *int) bool {
	parsed, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "id must be an integer"})
		return false
	}
	*id = parsed
	return true
}

func handleInternalError(w http.ResponseWriter, err error) {
	log.Printf("ERROR: %v", err)
	writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "An unexpected error occurred. Please try again later."})
}

func handleNotFoundError(w http.ResponseWriter) {
	writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "The requested post was not found."})
}

func (app *Application) handleCreatePost(w http.ResponseWriter, r *http.Request) {
	var body Post
	if !validateRequestBody(w, r, &body) {
		return
	}

	query := "INSERT INTO posts (title, content, category, tags) VALUES ($1, $2, $3, $4) RETURNING id, title, content, category, tags, created_at, updated_at"
	rows, err := app.pool.Query(r.Context(), query, body.Title, body.Content, body.Category, body.Tags)
	if err != nil {
		handleInternalError(w, err)
		return
	}

	post, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[Post])
	if err != nil {
		handleInternalError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, post)
}

func (app *Application) handleUpdatePost(w http.ResponseWriter, r *http.Request) {
	var id int
	var body Post
	if !validateRequestBody(w, r, &body) || !validateIDPathParam(w, r, &id) {
		return
	}

	query := "UPDATE posts SET title = $1, content = $2, category = $3, tags = $4, updated_at = NOW() WHERE id = $5 RETURNING id, title, content, category, tags, created_at, updated_at"
	rows, err := app.pool.Query(r.Context(), query, body.Title, body.Content, body.Category, body.Tags, id)
	if err != nil {
		handleInternalError(w, err)
		return
	}

	post, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[Post])
	if errors.Is(err, pgx.ErrNoRows) {
		handleNotFoundError(w)
		return
	} else if err != nil {
		handleInternalError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, post)
}

func (app *Application) handleDeletePost(w http.ResponseWriter, r *http.Request) {
	var id int
	if !validateIDPathParam(w, r, &id) {
		return
	}

	query := "DELETE FROM posts WHERE id = $1"
	result, err := app.pool.Exec(r.Context(), query, id)
	if err != nil {
		handleInternalError(w, err)
		return
	}

	if result.RowsAffected() == 0 {
		handleNotFoundError(w)
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}

func (app *Application) handleGetPost(w http.ResponseWriter, r *http.Request) {
	var id int
	if !validateIDPathParam(w, r, &id) {
		return
	}

	query := "SELECT * FROM posts WHERE id = $1"
	rows, err := app.pool.Query(r.Context(), query, id)
	if err != nil {
		handleInternalError(w, err)
		return
	}

	post, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[Post])
	if errors.Is(err, pgx.ErrNoRows) {
		handleNotFoundError(w)
		return
	} else if err != nil {
		handleInternalError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, post)
}

func (app *Application) handleGetPosts(w http.ResponseWriter, r *http.Request) {
	query := "SELECT * FROM posts"
	rows, err := app.pool.Query(r.Context(), query)
	if err != nil {
		handleInternalError(w, err)
		return
	}

	posts, err := pgx.CollectRows(rows, pgx.RowToStructByName[Post])
	if err != nil {
		handleInternalError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, posts)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	if data == nil {
		w.WriteHeader(status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		log.Printf("ERROR: %v", err)
		return
	}
}

func createPool() *pgxpool.Pool {
	ctx := context.Background()
	connStr := os.Getenv("DATABASE_URL")

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v\n", err)
	}

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Ping failed: %v\n", err)
	}

	return pool
}

func main() {
	app := &Application{pool: createPool()}
	defer app.pool.Close()

	mux := http.NewServeMux()

	mux.HandleFunc("POST /posts", app.handleCreatePost)
	mux.HandleFunc("PUT /posts/{id}", app.handleUpdatePost)
	mux.HandleFunc("DELETE /posts/{id}", app.handleDeletePost)
	mux.HandleFunc("GET /posts/{id}", app.handleGetPost)
	mux.HandleFunc("GET /posts", app.handleGetPosts)

	log.Print("Server is running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
