package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Short: "Redis Rate Limiter",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Redis Rate Limiter is running...")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
