package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "lnmonja",
	Short: "lnmonja CLI",
	Long:  "Command-line interface for lnmonja monitoring system",
}

var (
	serverAddr string
	apiKey     string
)

func main() {
	rootCmd.PersistentFlags().StringVar(&serverAddr, "server", "localhost:8080", "Server address")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key for authentication")

	rootCmd.AddCommand(
		NewNodesCommand(),
		NewMetricsCommand(),
		NewAlertsCommand(),
		NewConfigCommand(),
		NewStatusCommand(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func NewNodesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nodes",
		Short: "Manage nodes",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List all nodes",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("Listing nodes...")
				// Implementation
			},
		},
		&cobra.Command{
			Use:   "info [node-id]",
			Short: "Show node info",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Printf("Showing info for node: %s\n", args[0])
			},
		},
	)

	return cmd
}

func NewMetricsCommand() *cobra.Command {
	var query string
	var from, to string
	var step string

	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "Query metrics",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Querying metrics: %s\n", query)
			// Implementation
		},
	}

	cmd.Flags().StringVarP(&query, "query", "q", "", "PromQL query")
	cmd.Flags().StringVar(&from, "from", "1h", "Start time")
	cmd.Flags().StringVar(&to, "to", "now", "End time")
	cmd.Flags().StringVar(&step, "step", "15s", "Step interval")
	cmd.MarkFlagRequired("query")

	return cmd
}

func NewAlertsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alerts",
		Short: "Manage alerts",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List alerts",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("Listing alerts...")
			},
		},
		&cobra.Command{
			Use:   "silence [alert-id]",
			Short: "Silence an alert",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Printf("Silencing alert: %s\n", args[0])
			},
		},
	)

	return cmd
}

func NewStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show system status",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("=== lnmonja Status ===")
			fmt.Println("Server: Healthy")
			fmt.Println("Nodes: 5 connected")
			fmt.Println("Alerts: 0 firing")
			fmt.Println("Storage: 1.2GB used")
		},
	}

	return cmd
}

func NewConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "show",
			Short: "Show current config",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("Current configuration:")
				// Implementation
			},
		},
		&cobra.Command{
			Use:   "reload",
			Short: "Reload configuration",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("Reloading configuration...")
			},
		},
	)

	return cmd
}