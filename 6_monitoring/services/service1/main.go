package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
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

type LastOrderedProduct struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price"`
}

type User struct {
	ID                 int    `json:"id"`
	Username           string `json:"username"`
	Email              string `json:"email"`
	LastOrderedProduct int    `json:"last_ordered_product"`
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

func main() {
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

	http.HandleFunc("/users", logRequests(serviceLogWriter, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getUsers(db, w, r)
		case http.MethodPost:
			createUser(db, w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "Unsupported request method.")
		}
	}))

	http.HandleFunc("/users/", logRequests(serviceLogWriter, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getUser(db, w, r)
		case http.MethodPut:
			updateUser(db, w, r)
		case http.MethodDelete:
			deleteUser(db, w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "Unsupported request method.")
		}
	}))

	http.HandleFunc("/users/product/", logRequests(serviceLogWriter, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getLastOrderedProduct(db, w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "Unsupported request method.")
		}
	}))

	// Start HTTP server
	log.Fatal(http.ListenAndServe(":8000", nil))
}

func initKafkaWriter() *kafka.Writer {
	kafkaWriter := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  []string{"localhost:9092"},
		Topic:    "service-log",
		Balancer: &kafka.LeastBytes{},
	})
	return kafkaWriter
}

func logRequests(serviceLogWriter *kafka.Writer, next http.HandlerFunc) http.HandlerFunc {
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

func getUsers(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	timer := prometheus.NewTimer(dbQueryDuration)
	rows, err := db.Query("SELECT * FROM users")
	timer.ObserveDuration()
	if err != nil {
		http.Error(w, "Failed to query users", http.StatusInternalServerError)
		return
	}

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.LastOrderedProduct)
		if err != nil {
			http.Error(w, "Failed to scan user", http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func getUser(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/users/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid user ID.")
		return
	}

	timer := prometheus.NewTimer(dbQueryDuration)
	row := db.QueryRow("SELECT * FROM users WHERE id = $1", id)
	timer.ObserveDuration()

	var user User
	if err := row.Scan(&user.ID, &user.Username, &user.Email, &user.LastOrderedProduct); err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "User not found.")
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Failed to get user from the database: %s", err.Error())
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func createUser(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid user data.")
		return
	}

	if user.Username == "" || user.Email == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid user data.")
		return
	}

	timer := prometheus.NewTimer(dbQueryDuration)
	err := db.QueryRow("INSERT INTO users (username, email, last_ordered_product) VALUES ($1, $2, $3) RETURNING id", user.Username, user.Email, user.LastOrderedProduct).Scan(&user.ID)
	timer.ObserveDuration()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to create user in the database: %s", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func updateUser(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/users/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid user ID.")
		return
	}

	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid user data.")
		return
	}

	if user.Username == "" || user.Email == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid user data.")
		return
	}

	timer := prometheus.NewTimer(dbQueryDuration)
	result, err := db.Exec("UPDATE users SET username = $1, email = $2 WHERE id = $3", user.Username, user.Email, id)
	timer.ObserveDuration()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to update user in the database: %s", err.Error())
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "User not found.")
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "User updated successfully.")
}

func deleteUser(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/users/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid user ID.")
		return
	}

	timer := prometheus.NewTimer(dbQueryDuration)
	result, err := db.Exec("DELETE FROM users WHERE id = $1", id)
	timer.ObserveDuration()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to delete user from the database: %s", err.Error())
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "User not found.")
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "User deleted successfully.")
}

func getLastOrderedProduct(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/users/product/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid user ID.")
		return
	}
	fmt.Printf("Parsed user ID: %d\n", id)

	timer := prometheus.NewTimer(dbQueryDuration)
	row := db.QueryRow("SELECT * FROM users WHERE id = $1", id)
	timer.ObserveDuration()

	var user User
	if err := row.Scan(&user.ID, &user.Username, &user.Email, &user.LastOrderedProduct); err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "User not found.")
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Failed to get user from the database: %s", err.Error())
		}
		return
	}
	fmt.Printf("Fetched user: %#v\n", user)

	productsServiceName := os.Getenv("HELPER_SERVICE")
	url := fmt.Sprintf("http://%s/products/%d", productsServiceName, user.LastOrderedProduct)
	fmt.Printf("Sending request to URL: %s\n", url)
	resp, err := http.Get(url)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to delete user from the database: %s", err.Error())
		return
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Got response from another service: ", string(body))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(string(body))
}
