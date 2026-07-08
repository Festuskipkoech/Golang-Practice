package store

import (
	"errors"
	"go-crud-api/models"
	"sync"
)

// BookStore is our fake database — just a map living in memory.
// When the server restarts, data resets. Perfect for learning.
type BookStore struct {
	mu sync.RWMutex
	books map[string]models.Book
}
func NewBookStore() *BookStore {
	return &BookStore{
		books: map[string]models.Book{
			"1": {ID: "1", Title: "The Go programming language", Author: "Alan Donovan", Pages: 380, Price: 39.99},
            "2": {ID: "2", Title: "Clean Code", Author: "Robert Martin", Pages: 431, Price: 29.99},
            "3": {ID: "3", Title: "Designing Data-Intensive Applications", Author: "Martin Kleppmann", Pages: 562, Price: 49.99},
        },
	}
}

func (s *BookStore) GetAll() []models.Book {
	// RLock allows multiple readers simultaneously — no writer can write while this runs
	s.mu.RLock()
	defer s.mu.RUnlock()

	books := make([]models.Book, 0, len(s.books))
	for _, b := range s.books {
		books = append(books, b)
	}
	return books
}

func (s *BookStore) GetById(id string) (*models.Book, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	book, exists := s.books[id]
	if !exists {
		return nil, errors.New("book not found")
	}

	return &book, nil
}

func (s *BookStore) Create(book models.Book) models.Book {
	s.mu.Lock() // full lock — no readers or writers allowed during write
	defer s.mu.Unlock()

	s.books[book.ID] = book
	return book
}

func (s *BookStore) Update(id string, updated models.Book) (models.Book, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.books[id]; !exists {
		return  models.Book{}, errors.New("book not found")
	}
	updated.ID = id
	s.books[id] = updated
	return updated, nil
}

func (s *BookStore) Delete (id string) error{
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.books[id]; !exists {
		return  errors.New("book not found")
	}
	delete(s.books, id)
	return  nil
}
