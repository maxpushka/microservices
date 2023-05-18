package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type Product struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price"`
}

var products []Product

func main() {
	router := mux.NewRouter()

	router.HandleFunc("/products", getProducts).Methods("GET")
	router.HandleFunc("/products", createProduct).Methods("POST")
	router.HandleFunc("/products/{id}", getProduct).Methods("GET")
	router.HandleFunc("/products/{id}", updateProduct).Methods("PUT")
	router.HandleFunc("/products/{id}", deleteProduct).Methods("DELETE")

	log.Fatal(http.ListenAndServe(":8000", router))
}

func getProducts(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(products)
}

func createProduct(w http.ResponseWriter, r *http.Request) {
	var product Product
	_ = json.NewDecoder(r.Body).Decode(&product)
	products = append(products, product)
	json.NewEncoder(w).Encode(product)
}

func getProduct(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	for _, product := range products {
		if product.ID == params["id"] {
			json.NewEncoder(w).Encode(product)
			return
		}
	}
	json.NewEncoder(w).Encode(nil)
}

func updateProduct(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	for index, product := range products {
		if product.ID == params["id"] {
			products = append(products[:index], products[index+1:]...)
			var updatedProduct Product
				_ = json.NewDecoder(r.Body).Decode(&updatedProduct)
			updatedProduct.ID = params["id"]
			products = append(products, updatedProduct)
			json.NewEncoder(w).Encode(updatedProduct)
			return
		}
	}
	json.NewEncoder(w).Encode(nil)
}

func deleteProduct(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	for index, product := range products {
		if product.ID == params["id"] {
			products = append(products[:index], products[index+1:]...)
			break
		}
	}
	json.NewEncoder(w).Encode(products)
}
