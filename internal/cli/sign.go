package cli

import "github.com/spf13/cobra"

func cmdSign() *cobra.Command {
	c := &cobra.Command{
		Use:   "sign",
		Short: "Helpers for HTTP Message Signatures",
	}
	c.AddCommand(cmdSignCurl())
	return c
}
