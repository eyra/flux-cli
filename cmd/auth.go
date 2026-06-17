package cmd

import (
	"fmt"

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

		creds, err := auth.Load(env)
		if err != nil || creds == nil {
			fmt.Printf("Not signed in (%s)\n", env)
			fmt.Println("Run 'flux auth login' to sign in.")
			return nil
		}

		fmt.Printf("Signed in (%s)\n", env)
		fmt.Printf("Token expires: %s\n", creds.ExpiresAt.Format("2006-01-02 15:04"))
		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd, authLogoutCmd, authStatusCmd)
	rootCmd.AddCommand(authCmd)
}
