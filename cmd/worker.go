package cmd

import (
	"fmt"

	"github.com/Trishvan/queuectl/internal/worker"
	"github.com/spf13/cobra"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Manage worker processes",
}

var workerStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start one or more workers",
	Run: func(cmd *cobra.Command, args []string) {
		count, _ := cmd.Flags().GetInt("count")
		manager := worker.NewManager(count, db, cfg)
		manager.Start()
	},
}

var workerStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop running workers gracefully",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := worker.StopWorkers(); err != nil {
			return fmt.Errorf("failed to stop workers: %w", err)
		}
		return nil
	},
}

func init() {
	workerStartCmd.Flags().IntP("count", "c", 1, "Number of workers to start")
	workerCmd.AddCommand(workerStartCmd)
	workerCmd.AddCommand(workerStopCmd)
}
