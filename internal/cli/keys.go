package cli

import "github.com/spf13/cobra"

func cmdKeys() *cobra.Command {
	c := &cobra.Command{
		Use:   "keys",
		Short: "Key management",
	}
	c.AddCommand(cmdKeysNew())
	return c
}
