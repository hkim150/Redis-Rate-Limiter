package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	FixedWindowMaxRequests int64
	FixedWindowTTL         time.Duration
	TokenBucketMaxTokens   int
	TokenBucketRefillRate  float64
	TokenBucketTTL         int
}

type Server struct {
	redisClient *redis.Client
	config      *Config
	httpServer  *http.Server
}

func NewServer(redisClient *redis.Client, config *Config) *Server {
	return &Server{
		redisClient: redisClient,
		config:      config,
	}
}

// loggingMiddleware logs HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Create a response writer wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(wrapped, r)
		
		duration := time.Since(start)
		log.Printf("%s %s %d %v %s", r.Method, r.URL.Path, wrapped.statusCode, duration, r.RemoteAddr)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (s *Server) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// Check Redis connectivity
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	
	if err := s.redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Health check failed - Redis connection error: %v", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(fmt.Sprintf("Service unavailable - Redis connection failed: %v\n", err)))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Server is running and Redis is connected\n"))
}

func (s *Server) fixedWindowHandler(w http.ResponseWriter, r *http.Request) {
	// Get client identifier (could be IP, user ID, etc.)
	clientID := r.Header.Get("X-Client-ID")
	if clientID == "" {
		clientID = r.RemoteAddr
	}
	
	key := fmt.Sprintf("fixed_window:%s", clientID)
	
	count, err := s.redisClient.Incr(r.Context(), key).Result()
	if err != nil {
		log.Printf("Failed to increment counter: %v", err)
		http.Error(w, "Failed to increment counter: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if count == 1 {
		// Set expiration for the key if it's the first request
		err = s.redisClient.ExpireNX(r.Context(), key, s.config.FixedWindowTTL).Err()
		if err != nil {
			log.Printf("Failed to set expiration: %v", err)
			http.Error(w, "Failed to set expiration: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else if count > s.config.FixedWindowMaxRequests {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(fmt.Sprintf("BLOCK - count: %d\n", count)))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("ALLOW - count: %d\n", count)))
}

var luaScript = redis.NewScript(`
	local key_tokens = KEYS[1]
	local key_timestamp = KEYS[2]
	local max_tokens = tonumber(ARGV[1])
	local refill_rate = tonumber(ARGV[2])
	local now = tonumber(ARGV[3])

	local tokens = tonumber(redis.call("GET", key_tokens)) or max_tokens
	local last_refill = tonumber(redis.call("GET", key_timestamp)) or now

	local elapsed = now - last_refill
	local refill = elapsed * refill_rate
	tokens = math.min(max_tokens, tokens + refill)

	if tokens < 1 then
		return 0
	end

	redis.call("SET", key_tokens, tokens-1)
	redis.call("SET", key_timestamp, now)

	return tokens
`)

func (s *Server) tokenBucketHandler(w http.ResponseWriter, r *http.Request) {
	// Get client identifier (could be IP, user ID, etc.)
	clientID := r.Header.Get("X-Client-ID")
	if clientID == "" {
		clientID = r.RemoteAddr
	}
	
	now := time.Now().Unix()
	keyTokens := fmt.Sprintf("token_bucket:tokens:%s", clientID)
	keyTimestamp := fmt.Sprintf("token_bucket:timestamp:%s", clientID)

	token, err := luaScript.Run(r.Context(), s.redisClient, []string{keyTokens, keyTimestamp}, 
		s.config.TokenBucketMaxTokens, s.config.TokenBucketRefillRate, now).Int()
	if err != nil {
		log.Printf("Failed to execute lua script: %v", err)
		http.Error(w, "Failed to execute lua script in redis: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if token == 0 {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(fmt.Sprintf("BLOCK - Token count: %d\n", token)))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("ALLOW - Token count: %d\n", token)))
}

func (s *Server) Run(port string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.healthCheckHandler)
	mux.HandleFunc("/fixed-window", s.fixedWindowHandler)
	mux.HandleFunc("/token-bucket", s.tokenBucketHandler)

	// Apply logging middleware
	handler := s.loggingMiddleware(mux)

	s.httpServer = &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel to listen for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on port %s", port)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-stop
	log.Println("Shutting down server...")

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server gracefully
	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Printf("Error during server shutdown: %v", err)
	} else {
		log.Println("Server stopped gracefully")
	}
}

func Run(port string) {
	RunWithConfig(port, "redis:6379", "", 0)
}

func RunWithConfig(port, redisAddr, redisPassword string, redisDB int) {
	// Create Redis client with provided configuration
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
		Protocol: 2, // use RESP2 protocol
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Successfully connected to Redis")

	// Create configuration with default values
	config := &Config{
		FixedWindowMaxRequests: 3,
		FixedWindowTTL:         10 * time.Second,
		TokenBucketMaxTokens:   5,
		TokenBucketRefillRate:  1.0, // 1 token per second
		TokenBucketTTL:         60,  // 60 seconds
	}

	server := NewServer(redisClient, config)
	server.Run(port)
}
