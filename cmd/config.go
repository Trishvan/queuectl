package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		switch key {
		case "max-retries":
			retries, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("invalid value for max-retries: %s", value)
			}
			cfg.MaxRetries = retries
		case "backoff-base":
			base, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("invalid value for backoff-base: %s", value)
			}
			cfg.BackoffBase = base
		default:
			return fmt.Errorf("unknown configuration key: %s", key)
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("Configuration updated: %s = %s\n", key, value)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
}
