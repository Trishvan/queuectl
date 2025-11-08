package cmd

import (
	"fmt"
	"os"

	"github.com/your-username/queuectl/internal/store"
	"github.com/your-username/queuectl/internal/worker"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show summary of all job states & active workers",
	RunE: func(cmd *cobra.Command, args []string) error {
		summary, err := db.GetStatusSummary()
		if err != nil {
			return fmt.Errorf("failed to get status summary: %w", err)
		}

		fmt.Println("Job Status Summary:")
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"State", "Count"})

		states := []store.JobState{store.StatePending, store.StateProcessing, store.StateCompleted, store.StateFailed, store.StateDead}
		for _, state := range states {
			count := 0
			if val, ok := summary[state]; ok {
				count = val
			}
			table.Append([]string{string(state), fmt.Sprintf("%d", count)})
		}
		table.Render()

		fmt.Println("\nWorker Status:")
		if worker.GetActiveWorkerCount() > 0 {
			fmt.Println("Workers are running.")
		} else {
			fmt.Println("Workers are not running.")
		}

		return nil
	},
}
