package server

import (
	"fmt"
	"net/http"
	"time"

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

func (s *Server) fixedWindowHandler(w http.ResponseWriter, r *http.Request) {
	var maxRequests int64 = 3
	ttl := time.Second * 10

	count, err := s.redisClient.Incr(r.Context(), "fixed_window_counter").Result()
	if err != nil {
		http.Error(w, "Failed to incr: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if count == 1 {
		// Set expiration for the key if it's the first request
		err = s.redisClient.ExpireNX(r.Context(), "fixed_window_counter", ttl).Err()
		if err != nil {
			http.Error(w, "Failed to expire: + err.Error()", http.StatusInternalServerError)
			return
		}
	} else if count > maxRequests {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("BLOCK - count: " + fmt.Sprintf("%d\n", count)))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ALLOW - count: " + fmt.Sprintf("%d\n", count)))
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

func (s *Server) tockenBucketHandler(w http.ResponseWriter, r *http.Request) {
	now := time.Now().Unix()
	keyTokens := "token_bucket:tokens"
	keyTimestamp := "token_bucket:timestamp"
	maxTokens := 3
	refillRate := 0.0 // tokens per second
	ttl := 3 // seconds

	token, err := luaScript.Run(r.Context(), s.redisClient, []string{keyTokens, keyTimestamp}, maxTokens, refillRate, now, ttl).Int()
	if err != nil {
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
	http.HandleFunc("/health", s.healthCheckHandler)
	http.HandleFunc("/fixed-window", s.fixedWindowHandler)
	http.HandleFunc("/token-bucket", s.tockenBucketHandler)

	fmt.Println("Starting server on :", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}

func Run(port string) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
		Protocol: 2,  // use RESP2 protocol
	})

	server := NewServer(redisClient)
	server.Run(port)
}
