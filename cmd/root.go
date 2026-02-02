package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	envFlag  string
	jsonFlag bool
)

func getEnv() string {
	// Flag takes precedence, then env var, then default
	if envFlag != "" && envFlag != "prod" {
		return envFlag
	}
	if env := os.Getenv("FLUX_ENV"); env != "" {
		return env
	}
	return "prod"
}

var rootCmd = &cobra.Command{
	Use:   "flux",
	Short: "Flux CLI - Project management from the command line",
	Long: `Flux CLI provides command-line access to Flux project management.

Environments:
  prod  - eyra-flux (default) - Eyra dev projects (Next, Feldspar)
  test  - eyra-flux-test - Flux dogfooding`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&envFlag, "env", "e", "prod", "Environment: prod or test")
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
}
