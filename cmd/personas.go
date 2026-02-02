package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/eyra/flux-cli/internal/api"
	"github.com/spf13/cobra"
)

var personasCmd = &cobra.Command{
	Use:   "personas",
	Short: "Manage personas",
}

var personasListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available personas",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient(getEnv())

		personas, err := client.ListPersonas()
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

		return nil
	},
}

func init() {
	rootCmd.AddCommand(personasCmd)
	personasCmd.AddCommand(personasListCmd)
}
