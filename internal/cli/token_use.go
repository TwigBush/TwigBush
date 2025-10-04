package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func cmdTokenUse() *cobra.Command {
	var label string
	var token string
	var method string
	var url string
	var data string

	c := &cobra.Command{
		Use:   "use",
		Short: "Call a Resource Server with a token",
		Example: "twigbush token use --label access0 --method POST " +
			"--url https://rs.example/checkout -d @body.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			// If token not passed, try ~/.twigbush/tokens/<label>.json with {"value":"..."}
			if token == "" && label != "" {
				val, _ := loadTokenFromDisk(label)
				token = val
			}
			if token == "" {
				return fmt.Errorf("no token provided. Use --token or --label with a saved token")
			}
			var body []byte
			if len(data) > 0 {
				if data[0] == '@' {
					p := data[1:]
					b, err := os.ReadFile(p)
					if err != nil {
						return err
					}
					body = b
				} else {
					body = []byte(data)
				}
			}
			headers := map[string]string{
				"Authorization": "GNAP " + token,
				"Accept":        "application/json",
			}
			resp, code, err := httpDoJSON(method, url, body, headers)
			if err != nil {
				return err
			}
			fmt.Printf("HTTP %d\n", code)
			return printJSON(resp)
		},
	}
	c.Flags().StringVar(&label, "label", "access0", "token label to use")
	c.Flags().StringVar(&token, "token", "", "token value (overrides label lookup)")
	c.Flags().StringVar(&method, "method", "POST", "HTTP method")
	c.Flags().StringVar(&url, "url", "", "target URL")
	c.Flags().StringVarP(&data, "data", "d", "", "request body. Use @file.json to read from file")
	_ = c.MarkFlagRequired("url")
	return c
}

func loadTokenFromDisk(label string) (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(dir + "/tokens/" + label + ".json")
	if err != nil {
		return "", err
	}
	// Very small parser to avoid extra deps
	// Expect {"value":"..."}
	type t struct {
		Value string `json:"value"`
	}
	var tok t
	_ = jsonUnmarshal(b, &tok)
	return tok.Value, nil
}
