package cli

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
)

func cmdGrantRequest() *cobra.Command {
	var file string
	var endpoint string

	c := &cobra.Command{
		Use:   "request",
		Short: "Post a GNAP grant request",
		RunE: func(cmd *cobra.Command, args []string) error {
			b, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			url := strings.TrimRight(asBaseURL, "/") + endpoint
			resp, code, err := httpDoJSON("POST", url, b, nil)
			if err != nil {
				return err
			}
			fmt.Printf("HTTP %d\n", code)
			return printJSON(resp)
		},
	}
	c.Flags().StringVarP(&file, "file", "f", "", "grant request JSON file")
	c.Flags().StringVar(&endpoint, "endpoint", "/grant", "AS path for grant requests")
	_ = c.MarkFlagRequired("file")
	return c
}

// Optional helper to guess default samples path
func samplesPath(p string) string {
	if path.IsAbs(p) {
		return p
	}
	return path.Join("samples", p)
}
