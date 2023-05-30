package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
)

type Product struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price"`
}

var (
	httpRequestsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests received",
	})

	httpRequestDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "http_request_duration_seconds",
		Help: "Time taken to process an HTTP request",
	})

	dbConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "db_connections",
		Help: "Current number of database connections",
	})

	dbQueryDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "db_query_duration_seconds",
		Help: "Time taken to process a database query",
	})
)

func main() { // Read database connection information from environment variables
	host := os.Getenv("DB_HOST")
	portStr := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	// Convert port string to integer
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatal("Invalid DB_PORT value:", err)
	}

	// Establish database connection
	dbInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	fmt.Println(dbInfo)
	db, err := sql.Open("postgres", dbInfo)
	if err != nil {
		log.Fatal(err)
	}
	dbConnections.Inc() // Increment dbConnections gauge
	defer func() {
		dbConnections.Dec() // Decrement dbConnections gauge
		db.Close()
	}()

	// Initialize Kafka writer
	serviceLogWriter := initKafkaWriter()

	// Initialize HTTP routes
	http.Handle("/metrics", promhttp.Handler())

	http.HandleFunc("/products", logRequests(serviceLogWriter, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getProducts(db, w, r)
		case http.MethodPost:
			createProduct(db, w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "Unsupported request method.")
		}
	}))

	http.HandleFunc("/products/", logRequests(serviceLogWriter, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getProduct(db, w, r)
		case http.MethodPut:
			updateProduct(db, w, r)
		case http.MethodDelete:
			deleteProduct(db, w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "Unsupported request method.")
		}
	}))

	// Start the HTTP server
	log.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func logRequests(kafkaWriter *kafka.Writer, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Notify broker
		go func() {
			logData := map[string]interface{}{
				"method": r.Method,
				"path":   r.URL.Path,
				"ip":     r.RemoteAddr,
			}
			logBytes, _ := json.Marshal(logData)
			kafkaWriter.WriteMessages(
				context.Background(),
				kafka.Message{
					Value: logBytes,
				},
			)
		}()

		httpRequestsTotal.Inc()
		timer := prometheus.NewTimer(httpRequestDuration)
		next(w, r)
		timer.ObserveDuration()
	}
}

func initKafkaWriter() *kafka.Writer {
	brokers := []string{os.Getenv("KAFKA_HOST")}
	topic := os.Getenv("KAFKA_TOPIC")
	return kafka.NewWriter(kafka.WriterConfig{
		Brokers:  brokers,
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	})
}

func getProducts(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	timer := prometheus.NewTimer(dbQueryDuration)
	rows, err := db.Query("SELECT * FROM products")
	timer.ObserveDuration()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to get products from the database: %s", err.Error())
		return
	}
	defer rows.Close()

	products := make([]Product, 0)
	for rows.Next() {
		var product Product
		if err := rows.Scan(&product.ID, &product.Name, &product.Price); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Failed to scan products from the database: %s", err.Error())
			return
		}
		products = append(products, product)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}

func getProduct(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/products/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid product ID.")
		return
	}

	timer := prometheus.NewTimer(dbQueryDuration)
	row := db.QueryRow("SELECT * FROM products WHERE id = $1", id)
	timer.ObserveDuration()

	var product Product
	if err := row.Scan(&product.ID, &product.Name, &product.Price); err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "Product not found.")
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Failed to get product from the database: %s", err.Error())
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}

func createProduct(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	var product Product
	if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid product data.")
		return
	}

	if product.Name == "" || product.Price <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid product data.")
		return
	}

	timer := prometheus.NewTimer(dbQueryDuration)
	err := db.QueryRow("INSERT INTO products (name, price) VALUES ($1, $2) RETURNING id", product.Name, product.Price).Scan(&product.ID)
	timer.ObserveDuration()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to create product in the database: %s", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}

func updateProduct(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/products/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid product ID.")
		return
	}

	var product Product
	if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid product data.")
		return
	}

	if product.Name == "" || product.Price <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid product data.")
		return
	}

	timer := prometheus.NewTimer(dbQueryDuration)
	result, err := db.Exec("UPDATE products SET name = $1, price = $2 WHERE id = $3", product.Name, product.Price, id)
	timer.ObserveDuration()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to update product in the database: %s", err.Error())
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Product not found.")
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Product updated successfully.")
}

func deleteProduct(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/products/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid product ID.")
		return
	}

	timer := prometheus.NewTimer(dbQueryDuration)
	result, err := db.Exec("DELETE FROM products WHERE id = $1", id)
	timer.ObserveDuration()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to delete product from the database: ", err.Error())
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Product not found.")
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Product deleted successfully.")
}
