package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"go-crud-api/middleware"
	"go-crud-api/models"
	"go-crud-api/store"

	"github.com/google/uuid"
)

// BookHandler holds a reference to the store.
// This is Go's version of dependency injection — no frameworks needed.

type BookHandler struct {
	store *store.BookStore
}

func NewBookHandler(store *store.BookStore) *BookHandler {
	return &BookHandler{ store:store }
}

// respond is a small helper to write JSON responses consistently.
// DRY principle — we would repeat this in every handler otherwise.
func respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError sends a structured error response.
func respondError(w http.ResponseWriter, status int, message string) {
	respond(w,status, map[string]string{"error": message})
}

// Get /books
func (h *BookHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	requestId :=middleware.GateRequestID(r)
	slog.Info("Fetching all books", "request_id", requestId)

	books := h.store.GetAll()
	respond(w, http.StatusOK, books)
}

// GET /books/{id}
func(h *BookHandler) GetById(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	book, err := h.store.GetById(id)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respond(w, http.StatusOK, book)
}

// POST /books
func (h *BookHandler) Create(w http.ResponseWriter, r *http.Request) {
	var book models.Book

    // Decode reads the request body and maps JSON fields to the struct.
    // If the body is malformed JSON, this returns an error.
	if err := json.NewDecoder(r.Body).Decode(&book); err != nil {
		respondError(w, http.StatusBadRequest, "invalid requets body")
		return
	}

	if book.Title == "" || book.Author == "" {
		respondError(w, http.StatusBadRequest, "title and author are required")
		return
	}

	book.ID = uuid.NewString() //generate a unique id
	created := h.store.Create(book)
	respond(w, http.StatusCreated, created)
}

// PUT /books/id
func (h *BookHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")	

	var book models.Book
	if err := json.NewDecoder(r.Body).Decode(&book); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updated, err := h.store.Update(id, book)
	if err !=nil {
		respondError(w, http.StatusBadRequest, "invalid requets body")
		return
	}
	respond(w, http.StatusOK, updated)
}

// DELETE
func (h *BookHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.store.Delete(id); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]string{"message": "book deleted"})
}