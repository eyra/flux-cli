package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/eyra/flux-cli/internal/api"
	"github.com/spf13/cobra"
)

var (
	// Epics flags
	epicMilestoneFlag   string
	epicCompletedFlag   bool
	epicTitleFlag       string
	epicDescriptionFlag string
	epicAssigneesFlag   string
	epicBranchFlag      string
	epicUnlinkFlag      bool
	epicContentFlag     string
	epicPersonaFlag     string
)

var epicsCmd = &cobra.Command{
	Use:   "epics",
	Short: "Manage epics",
}

var epicsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List epics",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

		opts := api.ListEpicsOptions{
			Milestone: epicMilestoneFlag,
			Completed: epicCompletedFlag,
			Project:   getProject(),
		}

		epics, err := client.ListEpics(opts)
		if err != nil {
			return err
		}

		if jsonFlag {
			data, _ := json.MarshalIndent(epics, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(epics) == 0 {
			fmt.Println("No epics found.")
			return nil
		}

		for _, epic := range epics {
			milestoneStr := ""
			if epic.Milestone != "" {
				milestoneStr = fmt.Sprintf(" [%s]", epic.Milestone)
			}
			fmt.Printf("%s  %s%s\n", epic.ID, epic.Title, milestoneStr)
		}

		return nil
	},
}

var epicsGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get epic details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

		epic, err := client.GetEpic(args[0], getProject())
		if err != nil {
			return err
		}

		if jsonFlag {
			data, _ := json.MarshalIndent(epic, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("# %s\n\n", epic.Title)
		fmt.Printf("ID: %s\n", epic.ID)
		if epic.Milestone != "" {
			fmt.Printf("Milestone: %s\n", epic.Milestone)
		}
		if epic.Branch != "" {
			fmt.Printf("Branch: %s\n", epic.Branch)
		}
		if len(epic.Assignees) > 0 {
			fmt.Printf("Assignees: %v\n", epic.Assignees)
		}

		if epic.Description != "" {
			fmt.Printf("\n## Description\n\n%s\n", epic.Description)
		}

		if len(epic.LinkedIssues) > 0 {
			fmt.Printf("\n## Linked Issues (%d)\n\n", len(epic.LinkedIssues))
			for _, issue := range epic.LinkedIssues {
				fmt.Printf("- %s  %s\n", issue.ID, issue.Title)
			}
		}

		if len(epic.Thread) > 0 {
			fmt.Printf("\n## Thread (%d comments)\n\n", len(epic.Thread))
			for _, comment := range epic.Thread {
				fmt.Printf("**%s** (%s):\n%s\n\n", comment.Author, comment.Date, comment.Content)
			}
		}

		return nil
	},
}

var epicsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new epic",
	RunE: func(cmd *cobra.Command, args []string) error {
		if epicTitleFlag == "" {
			return fmt.Errorf("--title is required")
		}

		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

		req := api.CreateEpicRequest{
			Title:       epicTitleFlag,
			Description: epicDescriptionFlag,
			Milestone:   epicMilestoneFlag,
			Assignees:   epicAssigneesFlag,
			Project:     getProject(),
		}

		epic, err := client.CreateEpic(req)
		if err != nil {
			return err
		}

		if jsonFlag {
			data, _ := json.MarshalIndent(epic, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Created epic %s: %s\n", epic.ID, epic.Title)
		return nil
	},
}

var epicsUpdateCmd = &cobra.Command{
	Use:   "update [id]",
	Short: "Update an existing epic",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

		req := api.UpdateEpicRequest{
			Title:       epicTitleFlag,
			Description: epicDescriptionFlag,
			Milestone:   epicMilestoneFlag,
			Branch:      epicBranchFlag,
			Project:     getProject(),
		}

		epic, err := client.UpdateEpic(args[0], req)
		if err != nil {
			return err
		}

		if jsonFlag {
			data, _ := json.MarshalIndent(epic, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Updated epic %s: %s\n", epic.ID, epic.Title)
		return nil
	},
}

var epicsIssuesCmd = &cobra.Command{
	Use:   "issues [id]",
	Short: "List issues linked to an epic",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

		issues, err := client.ListEpicIssues(args[0], epicCompletedFlag, getProject())
		if err != nil {
			return err
		}

		if jsonFlag {
			data, _ := json.MarshalIndent(issues, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(issues) == 0 {
			fmt.Println("No issues linked to this epic.")
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

var epicsLinkCmd = &cobra.Command{
	Use:   "link [id]",
	Short: "Link an epic to a milestone",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if epicMilestoneFlag == "" {
			return fmt.Errorf("--milestone is required")
		}

		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

		action := "link"
		if epicUnlinkFlag {
			action = "unlink"
		}

		req := api.LinkEpicRequest{
			MilestoneID: epicMilestoneFlag,
			Action:      action,
			Project:     getProject(),
		}

		if err := client.LinkEpic(args[0], req); err != nil {
			return err
		}

		if jsonFlag {
			printOK("id", args[0])
		} else if epicUnlinkFlag {
			fmt.Printf("Unlinked epic %s from milestone %s\n", args[0], epicMilestoneFlag)
		} else {
			fmt.Printf("Linked epic %s to milestone %s\n", args[0], epicMilestoneFlag)
		}
		return nil
	},
}

var epicsCommentCmd = &cobra.Command{
	Use:   "comment [id]",
	Short: "Add a comment to an epic",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if epicContentFlag == "" {
			return fmt.Errorf("--content is required")
		}

		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

		req := api.CommentRequest{
			Content: epicContentFlag,
			Persona: epicPersonaFlag,
			Project: getProject(),
		}

		if err := client.AddEpicComment(args[0], req); err != nil {
			return err
		}

		if jsonFlag {
			printOK("id", args[0])
		} else {
			fmt.Printf("Added comment to epic %s\n", args[0])
		}
		return nil
	},
}

var epicsResyncCmd = &cobra.Command{
	Use:   "resync [id]",
	Short: "Refresh linked issue titles on an epic",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())
		result, err := client.ResyncEpic(args[0], getProject())
		if err != nil {
			return err
		}
		if jsonFlag {
			fmt.Printf(`{"ok":"true","updated":%d,"checked":%d}`+"\n", result.Updated, result.Checked)
		} else {
			fmt.Printf("Resynced: %d/%d linked issues updated\n", result.Updated, result.Checked)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(epicsCmd)
	epicsCmd.AddCommand(epicsListCmd)
	epicsCmd.AddCommand(epicsGetCmd)
	epicsCmd.AddCommand(epicsCreateCmd)
	epicsCmd.AddCommand(epicsUpdateCmd)
	epicsCmd.AddCommand(epicsIssuesCmd)
	epicsCmd.AddCommand(epicsLinkCmd)
	epicsCmd.AddCommand(epicsCommentCmd)
	epicsCmd.AddCommand(epicsResyncCmd)

	// List flags
	epicsListCmd.Flags().StringVar(&epicMilestoneFlag, "milestone", "", "Filter by milestone ID")
	epicsListCmd.Flags().BoolVar(&epicCompletedFlag, "completed", false, "Include completed epics")

	// Create flags
	epicsCreateCmd.Flags().StringVar(&epicTitleFlag, "title", "", "Epic title (required)")
	epicsCreateCmd.Flags().StringVar(&epicDescriptionFlag, "description", "", "Epic description")
	epicsCreateCmd.Flags().StringVar(&epicMilestoneFlag, "milestone", "", "Milestone ID to link to")
	epicsCreateCmd.Flags().StringVar(&epicAssigneesFlag, "assignees", "", "Comma-separated list of assignee IDs")

	// Update flags
	epicsUpdateCmd.Flags().StringVar(&epicTitleFlag, "title", "", "New title")
	epicsUpdateCmd.Flags().StringVar(&epicDescriptionFlag, "description", "", "New description")
	epicsUpdateCmd.Flags().StringVar(&epicMilestoneFlag, "milestone", "", "Milestone ID (empty to remove)")
	epicsUpdateCmd.Flags().StringVar(&epicBranchFlag, "branch", "", "Branch name (empty to remove)")

	// Issues flags
	epicsIssuesCmd.Flags().BoolVar(&epicCompletedFlag, "completed", false, "Include completed issues")

	// Link flags
	epicsLinkCmd.Flags().StringVar(&epicMilestoneFlag, "milestone", "", "Milestone ID (required)")
	epicsLinkCmd.Flags().BoolVar(&epicUnlinkFlag, "unlink", false, "Unlink instead of link")

	// Comment flags
	epicsCommentCmd.Flags().StringVar(&epicContentFlag, "content", "", "Comment content (required)")
	epicsCommentCmd.Flags().StringVar(&epicPersonaFlag, "persona", "", "Persona name for attribution")
}
