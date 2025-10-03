package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"time"

	"github.com/spf13/cobra"
)

// Starts AS and sample RS as child processes.
// Assumes you have ./cmd/as and ./cmd/rs packages in the repo.
func cmdRun() *cobra.Command {
	var port int
	var db string
	var fga string
	var logJSON bool

	c := &cobra.Command{
		Use:   "run",
		Short: "Start local AS and RS for dev",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

			as := exec.CommandContext(ctx, "go", "run", "./cmd/as")
			as.Env = append(os.Environ(),
				fmt.Sprintf("AS_PORT=%d", port),
				fmt.Sprintf("AS_DB=%s", db),
				fmt.Sprintf("FGA_ENDPOINT=%s", fga),
				fmt.Sprintf("LOG_JSON=%t", logJSON),
			)
			as.Stdout = os.Stdout
			as.Stderr = os.Stderr

			rs := exec.CommandContext(ctx, "go", "run", "./cmd/rs")
			rs.Env = append(os.Environ(),
				fmt.Sprintf("RS_PORT=%d", port+1),
				fmt.Sprintf("AS_BASE_URL=%s", asBaseURL),
				fmt.Sprintf("LOG_JSON=%t", logJSON),
			)
			rs.Stdout = os.Stdout
			rs.Stderr = os.Stderr

			fmt.Println("Starting AS...")
			if err := as.Start(); err != nil {
				return err
			}
			time.Sleep(800 * time.Millisecond)
			fmt.Println("Starting RS...")
			if err := rs.Start(); err != nil {
				return err
			}
			<-ctx.Done()
			fmt.Println("Shutting down...")
			return nil
		},
	}
	c.Flags().IntVar(&port, "port", 8089, "AS port, RS uses port+1")
	c.Flags().StringVar(&db, "db", "file:dev.db?_busy_timeout=5000&_fk=1", "AS database DSN")
	c.Flags().StringVar(&fga, "fga-endpoint", "", "OpenFGA endpoint URL")
	c.Flags().BoolVar(&logJSON, "log-json", false, "log in JSON format")
	return c
}
