package server

import (
	"fmt"
	"net/http"

	"github.com/redis/go-redis/v9"
)

type Server struct {
	redisClient *redis.Client
}

func NewServer(redisClient *redis.Client) *Server {
	return &Server{
		redisClient: redisClient,
	}
}

func (s *Server) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Server is running...\n"))
}

func (s *Server) rateLimitHandler(w http.ResponseWriter, r *http.Request) {
	// Implement rate limiting logic here using s.redisClient
	// For now, just respond with a placeholder message
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Rate limit handler not implemented yet.\n"))
}

func (s *Server) Run() {
	http.HandleFunc("/health", s.healthCheckHandler)
	http.HandleFunc("/rate-limit", s.rateLimitHandler)

	fmt.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}

func Run() {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
		Protocol: 2,  // use RESP2 protocol
	})

	server := NewServer(redisClient)
	server.Run()
}
