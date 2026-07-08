package models

// Book represents our data shape.
// The json tags tell Go how to serialize/deserialize field names.
// Without them, JSON would use "Title" instead of "title".

type Book struct {
	ID string `json:"id"`
	Title string `json:"title"`
	Author string `json:"author"`
	Pages int `json:"pages"`
	Price float64 `json:"price"`
}

