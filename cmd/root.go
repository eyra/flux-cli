package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/eyra/flux-cli/internal/auth"
	"github.com/spf13/cobra"
)

var (
	envFlag    string
	jsonFlag   bool
	apiKeyFlag string
	projectFlag string
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

func baseURLForEnv(env string) string {
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
	// Flag takes precedence, default to flux
	if projectFlag != "" {
		return projectFlag
	}
	return "flux"
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
	rootCmd.PersistentFlags().StringVar(&apiKeyFlag, "api-key", "", "API key for write operations (or use FLUX_API_KEY env var)")
	rootCmd.PersistentFlags().StringVar(&projectFlag, "project", "flux", "Project key: flux or next")
}
