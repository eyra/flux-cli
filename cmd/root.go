package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/eyra/flux-cli/internal/auth"
	"github.com/spf13/cobra"
)

var (
	envFlag     string
	jsonFlag    bool
	apiKeyFlag  string
	projectFlag string
)

func getEnv() string {
	// Flag takes precedence, then env var, then default
	if rootCmd.PersistentFlags().Changed("env") {
		return envFlag
	}
	if env := os.Getenv("FLUX_ENV"); env != "" {
		return env
	}
	return "prod"
}

func baseURLForEnv(env string) string {
	// FLUX_BASE_URL points the CLI at another server, e.g. a local one.
	if url := os.Getenv("FLUX_BASE_URL"); url != "" {
		return url
	}
	if env == "test" {
		return "https://eyra-flux-test.fly.dev"
	}
	return "https://eyra-flux.fly.dev"
}

func getAPIKey() string {
	if apiKeyFlag != "" {
		return apiKeyFlag
	}
	if key := os.Getenv("FLUX_API_KEY"); key != "" {
		return key
	}

	env := getEnv()
	creds, err := auth.Load(env)
	if err != nil || creds == nil {
		return ""
	}

	// Proactively refresh if expiring within 5 minutes
	if time.Until(creds.ExpiresAt) < 5*time.Minute {
		if creds.RefreshToken != "" {
			if newCreds, err := auth.Refresh(baseURLForEnv(env), creds.RefreshToken); err == nil {
				auth.Save(env, newCreds) //nolint:errcheck
				return newCreds.AccessToken
			}
		}
		return ""
	}

	return creds.AccessToken
}

func getProject() string {
	if projectFlag != "" {
		return projectFlag
	}
	if getEnv() == "test" {
		return "flux"
	}
	return "next"
}

func printOK(fields ...string) {
	if !jsonFlag {
		return
	}
	m := map[string]string{"ok": "true"}
	for i := 0; i+1 < len(fields); i += 2 {
		m[fields[i]] = fields[i+1]
	}
	data, _ := json.MarshalIndent(m, "", "  ")
	fmt.Println(string(data))
}

// printServerOK prints a server's JSON object response with the "ok" field
// and the given fields added, unless the server already sent them.
func printServerOK(response json.RawMessage, fields ...string) {
	m := map[string]interface{}{}
	if err := json.Unmarshal(response, &m); err != nil || m == nil {
		m = map[string]interface{}{}
	}
	m["ok"] = "true"
	for i := 0; i+1 < len(fields); i += 2 {
		if _, exists := m[fields[i]]; !exists {
			m[fields[i]] = fields[i+1]
		}
	}
	printJSON(m)
}

// printJSON prints v as indented JSON.
func printJSON(v interface{}) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

// removedCommand is a hidden command that only fails with message, so old
// scripts that still call it learn what to use instead. It accepts any
// arguments and flags.
func removedCommand(use, message string) *cobra.Command {
	return &cobra.Command{
		Use:                use,
		Short:              "Removed",
		Hidden:             true,
		DisableFlagParsing: true,
		SilenceUsage:       true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New(message)
		},
	}
}

func getAIModel(cmd *cobra.Command) *string {
	if !cmd.Flags().Changed("ai-model") {
		return nil
	}
	model, _ := cmd.Flags().GetString("ai-model")
	return &model
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
	rootCmd.PersistentFlags().StringVar(&apiKeyFlag, "api-key", "", "API key for API requests (or use FLUX_API_KEY env var)")
	rootCmd.PersistentFlags().StringVar(&projectFlag, "project", "", `Project key, e.g. flux, next or feldspar; "flux projects list" shows all (default: next on prod, flux on test)`)
}
