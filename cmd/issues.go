package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/eyra/flux-cli/internal/api"
	"github.com/spf13/cobra"
)

var stageFlag string

var issuesCmd = &cobra.Command{
	Use:   "issues",
	Short: "Manage issues",
}

var issuesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List issues",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient(envFlag)

		issues, err := client.ListIssues(stageFlag)
		if err != nil {
			return err
		}

		if jsonFlag {
			data, _ := json.MarshalIndent(issues, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		// Human-readable output
		if len(issues) == 0 {
			fmt.Println("No issues found.")
			return nil
		}

		for _, issue := range issues {
			stageStr := issue.Stage
			if issue.SubStage != "" {
				stageStr = fmt.Sprintf("%s > %s", issue.Stage, issue.SubStage)
			}
			fmt.Printf("%s  %s  [%s]\n", issue.ID, issue.Title, stageStr)
		}

		return nil
	},
}

var issuesGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get issue details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient(envFlag)

		issue, err := client.GetIssue(args[0])
		if err != nil {
			return err
		}

		if jsonFlag {
			data, _ := json.MarshalIndent(issue, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		// Human-readable output
		fmt.Printf("# %s\n\n", issue.Title)
		fmt.Printf("ID: %s\n", issue.ID)
		fmt.Printf("Stage: %s", issue.Stage)
		if issue.SubStage != "" {
			fmt.Printf(" > %s", issue.SubStage)
		}
		fmt.Println()

		if issue.Program != "" {
			fmt.Printf("Program: %s\n", issue.Program)
		}
		if issue.Size != "" {
			fmt.Printf("Size: %s\n", issue.Size)
		}

		fmt.Printf("\n## Description\n\n%s\n", issue.Description)

		if len(issue.Thread) > 0 {
			fmt.Printf("\n## Thread (%d comments)\n\n", len(issue.Thread))
			for _, comment := range issue.Thread {
				fmt.Printf("**%s** (%s):\n%s\n\n", comment.Author, comment.Date, comment.Content)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(issuesCmd)
	issuesCmd.AddCommand(issuesListCmd)
	issuesCmd.AddCommand(issuesGetCmd)

	issuesListCmd.Flags().StringVarP(&stageFlag, "stage", "s", "", "Filter by stage (specification, design, development, testing)")
}
