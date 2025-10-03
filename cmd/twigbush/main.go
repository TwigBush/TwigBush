package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var greetCmd = &cobra.Command{
	Use:   "greet [name]",
	Short: "Print a friendly greeting",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := "world"
		if len(args) == 1 {
			name = args[0]
		}
		fmt.Printf("Hello, %s!\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(greetCmd)
}
