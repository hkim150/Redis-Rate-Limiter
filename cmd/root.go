package cmd

import (
	"fmt"
	"os"
	"redisratelimiter/internal/server"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.Flags().StringP("port", "p", "8080", "port to run the server")
}

var rootCmd = &cobra.Command{
	Short: "Redis Rate Limiter",
	Run: func(cmd *cobra.Command, args []string) {
		port := cmd.Flag("port").Value.String()
		server.Run(port)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
