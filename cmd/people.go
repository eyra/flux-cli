package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/eyra/flux-cli/internal/api"
	"github.com/spf13/cobra"
)

var peopleCmd = &cobra.Command{
	Use:   "people",
	Short: "Manage people",
}

var peopleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List people on the project",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())
		people, err := client.ListPeople(getProject())
		if err != nil {
			return err
		}

		if jsonFlag {
			data, _ := json.MarshalIndent(people, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		for _, p := range people {
			fmt.Printf("%d  %s\n", p.ID, p.Name)
		}
		return nil
	},
}

func init() {
	peopleCmd.AddCommand(peopleListCmd)
	rootCmd.AddCommand(peopleCmd)
}
