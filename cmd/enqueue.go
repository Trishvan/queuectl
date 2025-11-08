package cmd

import (
	"fmt"

	"github.com/your-username/queuectl/internal/store"
	"github.com/spf13/cobra"
)

var enqueueCmd = &cobra.Command{
	Use:   "enqueue <json_spec>",
	Short: "Add a new job to the queue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jobSpec := args[0]
		job, err := store.NewJobFromSpec(jobSpec, cfg.MaxRetries)
		if err != nil {
			return fmt.Errorf("invalid job spec: %w", err)
		}

		if err := db.Enqueue(job); err != nil {
			return fmt.Errorf("failed to enqueue job: %w", err)
		}

		fmt.Printf("Successfully enqueued job with ID: %s\n", job.ID)
		return nil
	},
}
