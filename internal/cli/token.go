package cli

import "github.com/spf13/cobra"

func cmdToken() *cobra.Command {
	c := &cobra.Command{
		Use:   "token",
		Short: "Use or inspect tokens",
	}
	c.AddCommand(cmdTokenUse())
	return c
}
