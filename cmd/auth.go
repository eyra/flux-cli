package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/eyra/flux-cli/internal/api"
	"github.com/eyra/flux-cli/internal/auth"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Sign in to Flux",
	RunE: func(cmd *cobra.Command, args []string) error {
		env := getEnv()
		baseURL := baseURLForEnv(env)

		creds, err := auth.Login(baseURL)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		if err := auth.Save(env, creds); err != nil {
			return fmt.Errorf("failed to save credentials: %w", err)
		}

		fmt.Println("✓ Signed in successfully")
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Sign out from Flux",
	RunE: func(cmd *cobra.Command, args []string) error {
		env := getEnv()

		if err := auth.Clear(env); err != nil {
			return fmt.Errorf("failed to clear credentials: %w", err)
		}

		fmt.Printf("Signed out (%s)\n", env)
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		env := getEnv()

		client := api.NewClient(baseURLForEnv(env), getAccessToken())
		identity, err := client.GetIdentity()
		if err != nil {
			return err
		}

		if jsonFlag {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(struct {
				SignedIn bool   `json:"signed_in"`
				Env      string `json:"env"`
				*api.Identity
			}{
				SignedIn: true,
				Env:      env,
				Identity: identity,
			})
		} else {
			cmd.Printf("Signed in as %s (%s)\n", identity.DisplayName, env)
			cmd.Printf("Basecamp account: %s\n", identity.BasecampAccountID)
			cmd.Printf("Basecamp person: %s\n", identity.BasecampPersonID)
		}
		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd, authLogoutCmd, authStatusCmd)
	rootCmd.AddCommand(authCmd)
}
