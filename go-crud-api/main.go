package main

import (
	"fmt"
	"log"
	"net/http"

	"go-crud-api/handlers"
	"go-crud-api/store"
)

func main() {
	// Build the dependency chain bottom-up:
    // store -> handler -> router -> server
	bookStore := store.NewBookStore()
	bookHander := handlers.NewBookHandler(bookStore)

	mux := http.NewServeMux()

	// method path routing
	mux.HandleFunc("GET /books", bookHander.GetAll)
	mux.HandleFunc("GET /books/{id}",bookHander.GetById)
	mux.HandleFunc("POST /books", bookHander.Create)
	mux.HandleFunc("PUT /books/{id}", bookHander.Update)
	mux.HandleFunc("DELETE /books/{id}", bookHander.Delete)

	port := ":8080"
	fmt.Printf("Server running on http://localhost%s\n", port)

	// ListenAndServe blocks forever — it's the event loop.
    // log.Fatal prints the error and exits if the server fails to start.
	log.Fatal(http.ListenAndServe(port, mux))
}