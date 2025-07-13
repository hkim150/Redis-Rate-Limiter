package cmd

import (
	"log"
	"os"
	"redisratelimiter/internal/server"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.Flags().StringP("port", "p", "8080", "Port to run the server on")
	rootCmd.Flags().StringP("redis-addr", "r", "redis:6379", "Redis server address")
	rootCmd.Flags().StringP("redis-password", "", "", "Redis password (if required)")
	rootCmd.Flags().IntP("redis-db", "d", 0, "Redis database number")
}

var rootCmd = &cobra.Command{
	Use:   "redis-rate-limiter",
	Short: "A distributed rate limiter using Redis",
	Long: `A high-performance distributed rate limiter that supports multiple algorithms:
- Fixed Window: Simple counter-based rate limiting
- Token Bucket: Smooth rate limiting with burst capacity

The service provides HTTP endpoints for rate limiting decisions and can be used
as a microservice in distributed systems.`,
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetString("port")
		redisAddr, _ := cmd.Flags().GetString("redis-addr")
		redisPassword, _ := cmd.Flags().GetString("redis-password")
		redisDB, _ := cmd.Flags().GetInt("redis-db")
		
		log.Printf("Starting Redis Rate Limiter with config:")
		log.Printf("  Port: %s", port)
		log.Printf("  Redis Address: %s", redisAddr)
		log.Printf("  Redis DB: %d", redisDB)
		
		server.RunWithConfig(port, redisAddr, redisPassword, redisDB)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Printf("Error executing command: %v", err)
		os.Exit(1)
	}
}
