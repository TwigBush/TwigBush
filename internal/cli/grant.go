package cli

import "github.com/spf13/cobra"

func cmdGrant() *cobra.Command {
	c := &cobra.Command{
		Use:   "grant",
		Short: "GNAP grant operations",
	}
	c.AddCommand(cmdGrantRequest())
	return c
}
