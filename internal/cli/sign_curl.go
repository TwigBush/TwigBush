package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Stub that preserves UX. Wire real signing later.
func cmdSignCurl() *cobra.Command {
	var keyPath string
	var method string
	var url string
	var httpsig bool

	c := &cobra.Command{
		Use:   "curl",
		Short: "Wrap a curl with HTTP Message Signatures",
		Example: "twigbush sign curl --httpsig --key ~/.twigbush/keys/key-XYZ.jwk " +
			"--method POST --url https://api.example/checkout -d @body.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !httpsig {
				return fmt.Errorf("choose one: --httpsig")
			}
			_ = keyPath // placeholder for future signer
			headers := map[string]string{}

			if httpsig {
				headers["Signature"] = "sig1=:placeholder:;keyid=\"thumbprint\";alg=\"ed25519\""
				headers["Signature-Input"] = "sig1=();created=0000000000"
			}
			cmdStr := curlFor(method, url, nil, headers)
			fmt.Println(cmdStr)
			fmt.Println("Note: this is a placeholder. Implement real signing when ready.")
			return nil
		},
	}
	c.Flags().StringVar(&keyPath, "key", "", "path to private JWK")
	c.Flags().StringVar(&method, "method", "GET", "HTTP method")
	c.Flags().StringVar(&url, "url", "", "target URL")
	c.Flags().BoolVar(&httpsig, "httpsig", false, "use HTTP Message Signatures")
	_ = c.MarkFlagRequired("url")
	_ = c.MarkFlagRequired("key")
	return c
}
