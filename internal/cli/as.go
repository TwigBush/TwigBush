package cli

import "github.com/spf13/cobra"

func cmdAS() *cobra.Command {
	c := &cobra.Command{
		Use:   "as",
		Short: "Authorization Server helpers",
	}
	c.AddCommand(cmdASIntrospect())
	return c
}
