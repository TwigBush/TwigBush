package cli

import (
	"fmt"

	"github.com/TwigBush/gnap-go/internal/version"
	"github.com/spf13/cobra"
)

func cmdVersion() *cobra.Command {
	var verbose bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			if verbose {
				fmt.Println(version.Verbose())
			} else {
				fmt.Println(version.String())
			}
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed version information")

	return cmd
}
