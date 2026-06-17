package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/eyra/flux-cli/internal/api"
	"github.com/spf13/cobra"
)

var (
	personaTypeFlag    string
	includePromptFlag  bool
)

var personasCmd = &cobra.Command{
	Use:   "personas",
	Short: "Manage personas",
}

var personasListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available personas",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

		opts := &api.ListPersonasOptions{
			Type:          personaTypeFlag,
			IncludePrompt: includePromptFlag,
		}

		personas, err := client.ListPersonas(opts)
		if err != nil {
			return err
		}

		if jsonFlag {
			data, _ := json.MarshalIndent(personas, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		// Human-readable output
		if len(personas) == 0 {
			fmt.Println("No personas found.")
			return nil
		}

		// Group by type if showing all
		if personaTypeFlag == "all" || personaTypeFlag == "" {
			fmt.Println("Dev Personas:")
			for _, p := range personas {
				if p.Type == "dev" {
					fmt.Printf("  %s - %s\n", p.Name, p.Role)
				}
			}

			fmt.Println("\nConversation Personas:")
			for _, p := range personas {
				if p.Type == "conversation" {
					fmt.Printf("  %s - %s\n", p.Name, p.Role)
				}
			}
		} else {
			for _, p := range personas {
				fmt.Printf("%s - %s\n", p.Name, p.Role)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(personasCmd)
	personasCmd.AddCommand(personasListCmd)

	personasListCmd.Flags().StringVarP(&personaTypeFlag, "type", "t", "all", "Filter by type: dev, conversation, or all")
	personasListCmd.Flags().BoolVar(&includePromptFlag, "include-prompt", false, "Include system prompt in output (JSON only)")
}
