package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/Trishvan/queuectl/internal/store"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List jobs by state",
	RunE: func(cmd *cobra.Command, args []string) error {
		stateStr, _ := cmd.Flags().GetString("state")
		state := store.JobState(strings.ToLower(stateStr))

		validStates := map[store.JobState]bool{
			store.StatePending: true, store.StateProcessing: true, store.StateCompleted: true, store.StateFailed: true, store.StateDead: true,
		}
		if !validStates[state] {
			return fmt.Errorf("invalid state: %s. valid states are pending, processing, completed, failed, dead", stateStr)
		}

		jobs, err := db.ListJobsByState(state)
		if err != nil {
			return fmt.Errorf("failed to list jobs: %w", err)
		}

		if len(jobs) == 0 {
			fmt.Printf("No jobs found in '%s' state.\n", state)
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

func init() {
	listCmd.Flags().String("state", "pending", "State of the jobs to list (pending, processing, completed, failed, dead)")
}
