package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	output    string
	showCurl  bool
	asBaseURL string
	rsBaseURL string
	cfgPath   string
)

var rootCmd = &cobra.Command{
	Use:   "twigbush",
	Short: "TwigBush developer CLI for GNAP flows",
}

func Execute() error { return rootCmd.Execute() }

func init() {
	home, _ := os.UserHomeDir()
	defaultCfg := filepath.Join(home, ".twigbush", "config.yaml")

	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "json", "output format: json|yaml|table")
	rootCmd.PersistentFlags().BoolVar(&showCurl, "show-curl", false, "print equivalent curl for networked commands")
	rootCmd.PersistentFlags().StringVar(&asBaseURL, "as-base-url", "http://localhost:8089", "Authorization Server base URL")
	rootCmd.PersistentFlags().StringVar(&rsBaseURL, "rs-base-url", "http://localhost:8089", "Resource Server base URL")
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", defaultCfg, "config file path")

	// Wire top level groups
	rootCmd.AddCommand(cmdInit(), cmdRun(), cmdKeys(), cmdSign(), cmdGrant(), cmdToken(), cmdAS())

	// Friendly hint on no args
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:   "help",
		Short: "Show help",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().Help()
		},
	})
	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		fmt.Println("Use -h for help, for example: twigbush grant request -f samples/grants/basic.json")
	}
}
