package cmd

import (
	"fmt"
	"os"

	"github.com/your-username/queuectl/internal/config"
	"github.com/your-username/queuectl/internal/store"
	"github.com/spf13/cobra"
)

var (
	cfg   *config.Config
	db    store.Store
	rootCmd = &cobra.Command{
		Use:   "queuectl",
		Short: "A CLI-based background job queue system",
		Long:  `queuectl is a tool to manage background jobs with workers, retries, and a dead letter queue.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			cfg, err = config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Don't open DB for config command
			if cmd.Parent() != nil && cmd.Parent().Name() == "config" || cmd.Name() == "config" {
				return nil
			}

			db, err = store.NewSQLiteStore(cfg.DatabasePath)
			if err != nil {
				return fmt.Errorf("failed to initialize database: %w", err)
			}
			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			if db != nil {
				return db.Close()
			}
			return nil
		},
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(enqueueCmd)
	rootCmd.AddCommand(workerCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(dlqCmd)
	rootCmd.AddCommand(configCmd)
}
