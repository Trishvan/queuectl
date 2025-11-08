package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/Trishvan/queuectl/internal/store"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var dlqCmd = &cobra.Command{
	Use:   "dlq",
	Short: "Manage the Dead Letter Queue (DLQ)",
}

var dlqListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all jobs in the DLQ",
	RunE: func(cmd *cobra.Command, args []string) error {
		jobs, err := db.ListJobsByState(store.StateDead)
		if err != nil {
			return fmt.Errorf("failed to list DLQ jobs: %w", err)
		}

		if len(jobs) == 0 {
			fmt.Println("Dead Letter Queue is empty.")
			return nil
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"ID", "Command", "Attempts", "Created At", "Updated At"})
		for _, job := range jobs {
			table.Append([]string{
				job.ID,
				job.Command,
				fmt.Sprintf("%d", job.Attempts),
				job.CreatedAt.Format("2006-01-02 15:04:05"),
				job.UpdatedAt.Format("2006-01-02 15:04:05"),
			})
		}
		table.Render()
		return nil
	},
}

var dlqRetryCmd = &cobra.Command{
	Use:   "retry <job_id>",
	Short: "Retry a job from the DLQ",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jobID := args[0]
		job, err := db.GetJob(jobID)
		if err != nil {
			return fmt.Errorf("failed to get job %s: %w", jobID, err)
		}

		if job.State != store.StateDead {
			return fmt.Errorf("job %s is not in the DLQ (current state: %s)", jobID, job.State)
		}

		// Reset job for retry
		job.State = store.StatePending
		job.Attempts = 0
		job.NextRunAt = time.Now().UTC()

		if err := db.UpdateJob(job); err != nil {
			return fmt.Errorf("failed to retry job %s: %w", jobID, err)
		}

		fmt.Printf("Job %s has been moved from DLQ back to the pending queue.\n", jobID)
		return nil
	},
}

func init() {
	dlqCmd.AddCommand(dlqListCmd)
	dlqCmd.AddCommand(dlqRetryCmd)
}
