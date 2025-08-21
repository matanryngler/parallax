package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

// Test data structures
type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type APIResponse struct {
	Data    []User `json:"data"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Test data
var testUsers = []User{
	{ID: 1, Name: "alice"},
	{ID: 2, Name: "bob"},
	{ID: 3, Name: "charlie"},
}

func main() {
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/simple-array", simpleArrayHandler)
	http.HandleFunc("/complex-json", complexJSONHandler)
	http.HandleFunc("/auth/basic", basicAuthHandler)
	http.HandleFunc("/auth/bearer", bearerAuthHandler)
	http.HandleFunc("/error/404", notFoundHandler)
	http.HandleFunc("/error/500", serverErrorHandler)
	http.HandleFunc("/slow", slowHandler)

	log.Println("🚀 API test server starting on :8080")
	log.Println("📋 Available endpoints:")
	log.Println("  GET /health - Health check")
	log.Println("  GET /simple-array - Simple JSON array")
	log.Println("  GET /complex-json - Complex nested JSON")
	log.Println("  GET /auth/basic - Basic auth required (user:pass)")
	log.Println("  GET /auth/bearer - Bearer token required (token123)")
	log.Println("  GET /error/404 - Returns 404")
	log.Println("  GET /error/500 - Returns 500")
	log.Println("  GET /slow - Slow response (5s delay)")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

func simpleArrayHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	simpleArray := []string{"api-item-1", "api-item-2", "api-item-3"}
	json.NewEncoder(w).Encode(simpleArray)
}

func complexJSONHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := APIResponse{
		Data:    testUsers,
		Status:  "success",
		Message: "Users retrieved successfully",
	}
	json.NewEncoder(w).Encode(response)
}

func basicAuthHandler(w http.ResponseWriter, r *http.Request) {
	username, password, ok := r.BasicAuth()
	if !ok || username != "testuser" || password != "testpass" {
		w.Header().Set("WWW-Authenticate", `Basic realm="Test Realm"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	authItems := []string{"auth-basic-1", "auth-basic-2"}
	json.NewEncoder(w).Encode(authItems)
}

func bearerAuthHandler(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "Missing Bearer token", http.StatusUnauthorized)
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token != "test-token-123" {
		http.Error(w, "Invalid Bearer token", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	bearerItems := []string{"auth-bearer-1", "auth-bearer-2"}
	json.NewEncoder(w).Encode(bearerItems)
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not Found", http.StatusNotFound)
}

func serverErrorHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

func slowHandler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(5 * time.Second)
	w.Header().Set("Content-Type", "application/json")
	slowItems := []string{"slow-item-1", "slow-item-2"}
	json.NewEncoder(w).Encode(slowItems)
}
