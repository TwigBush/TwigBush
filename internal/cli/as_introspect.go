package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func cmdASIntrospect() *cobra.Command {
	var token string
	var endpoint string

	c := &cobra.Command{
		Use:   "introspect",
		Short: "Call the AS introspection endpoint",
		RunE: func(cmd *cobra.Command, args []string) error {
			if token == "" {
				return fmt.Errorf("--token is required")
			}
			url := strings.TrimRight(asBaseURL, "/") + endpoint
			body := []byte(fmt.Sprintf(`{"token": %q}`, token))
			resp, code, err := httpDoJSON("POST", url, body, map[string]string{"Accept": "application/json"})
			if err != nil {
				return err
			}
			fmt.Printf("HTTP %d\n", code)
			return printJSON(resp)
		},
	}
	c.Flags().StringVar(&token, "token", "", "access token to introspect")
	c.Flags().StringVar(&endpoint, "endpoint", "/introspect", "AS path for introspection")
	_ = c.MarkFlagRequired("token")
	return c
}
