package main

import (
	"fmt"
	"log"
	"net/http"
)

func greetHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "Guest"
	}

	greeting := fmt.Sprintf("Hello, %s!", name)
	fmt.Fprintf(w, greeting)
}

func main() {
	http.HandleFunc("/", greetHandler)

	fmt.Println("Starting server on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
