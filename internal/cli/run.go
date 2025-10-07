package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

// Starts AS
func cmdRun() *cobra.Command {
	var port int
	var db string
	var fga string
	var logJSON bool

	c := &cobra.Command{
		Use:   "run",
		Short: "Start local AS and RS for dev",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Find the binary directory
			binDir, err := getBinaryDirectory()
			if err != nil {
				return fmt.Errorf("failed to locate binaries: %w", err)
			}

			// Construct paths to the binaries
			asBinary := filepath.Join(binDir, "as")
			rsBinary := filepath.Join(binDir, "playground")

			// Check if binaries exist
			if !fileExists(asBinary) {
				return fmt.Errorf("authorization server binary not found at %s. Run 'make build' first", asBinary)
			}
			if !fileExists(rsBinary) {
				return fmt.Errorf("playground binary not found at %s. Run 'make build' first", rsBinary)
			}

			// Start authorization server
			as := exec.CommandContext(ctx, asBinary)
			as.Env = append(os.Environ(),
				fmt.Sprintf("AS_PORT=%d", port),
				fmt.Sprintf("AS_DB=%s", db),
				fmt.Sprintf("FGA_ENDPOINT=%s", fga),
				fmt.Sprintf("LOG_JSON=%t", logJSON),
			)
			as.Stdout = os.Stdout
			as.Stderr = os.Stderr

			// Start resource server/playground
			rs := exec.CommandContext(ctx, rsBinary)
			rs.Env = append(os.Environ(),
				fmt.Sprintf("RS_PORT=%d", port+1),
				fmt.Sprintf("AS_BASE_URL=%s", asBaseURL),
				fmt.Sprintf("LOG_JSON=%t", logJSON),
			)
			rs.Stdout = os.Stdout
			rs.Stderr = os.Stderr

			fmt.Printf("Starting authorization server from: %s\n", asBinary)
			if err := as.Start(); err != nil {
				return fmt.Errorf("failed to start authorization server: %w", err)
			}

			fmt.Printf("Starting playground from: %s\n", rsBinary)
			if err := rs.Start(); err != nil {
				return fmt.Errorf("failed to start playground: %w", err)
			}

			// Wait for both processes
			errChan := make(chan error, 2)
			go func() { errChan <- as.Wait() }()
			go func() { errChan <- rs.Wait() }()

			return <-errChan
		},
	}
	c.Flags().IntVar(&port, "port", 8085, "AS port, RS uses port+1")
	c.Flags().StringVar(&db, "db", "file:dev.db?_busy_timeout=5000&_fk=1", "AS database DSN")
	c.Flags().StringVar(&fga, "fga-endpoint", "", "OpenFGA endpoint URL")
	c.Flags().BoolVar(&logJSON, "log-json", false, "log in JSON format")
	return c
}

// getBinaryDirectory finds the directory containing the compiled binaries
func getBinaryDirectory() (string, error) {
	// Get the directory of the currently running executable
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}

	exeDir := filepath.Dir(exe)

	// If running from dist/, use that directory
	if filepath.Base(exeDir) == "dist" {
		return exeDir, nil
	}

	// Otherwise, assume binaries are in dist/ relative to project root
	// This handles the case when the binary is installed or symlinked
	return filepath.Join(exeDir, "dist"), nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
