package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/eyra/flux-cli/internal/api"
	"github.com/spf13/cobra"
)

var (
	// Milestones flags
	milestoneCompletedFlag       bool
	milestoneTitleFlag           string
	milestoneDescriptionFlag     string
	milestoneRepoFlag            string
	milestoneBranchFlag          string
	milestoneWorkflowFlag        string
	milestoneGithubMilestoneFlag int
	milestoneAssigneesFlag       string
	milestoneContentFlag         string
	milestonePersonaFlag         string
)

var milestonesCmd = &cobra.Command{
	Use:   "milestones",
	Short: "Manage milestones",
}

var milestonesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List milestones",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

		opts := api.ListMilestonesOptions{
			Completed: milestoneCompletedFlag,
			Project:   getProject(),
		}

		milestones, err := client.ListMilestones(opts)
		if err != nil {
			return err
		}

		if jsonFlag {
			data, _ := json.MarshalIndent(milestones, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(milestones) == 0 {
			fmt.Println("No milestones found.")
			return nil
		}

		for _, milestone := range milestones {
			branchStr := ""
			if milestone.Branch != "" {
				branchStr = fmt.Sprintf(" [%s]", milestone.Branch)
			}
			completedStr := ""
			if milestone.Completed {
				completedStr = " [done]"
			}
			fmt.Printf("%s  %s%s%s\n", milestone.ID, milestone.Title, branchStr, completedStr)
		}

		return nil
	},
}

var milestonesGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get milestone details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

		milestone, err := client.GetMilestone(args[0], getProject())
		if err != nil {
			return err
		}

		if jsonFlag {
			data, _ := json.MarshalIndent(milestone, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("# %s\n\n", milestone.Title)
		fmt.Printf("ID: %s\n", milestone.ID)
		if milestone.Completed {
			fmt.Printf("Completed: yes\n")
		}
		if milestone.Repo != "" {
			fmt.Printf("Repo: %s\n", milestone.Repo)
		}
		if milestone.Branch != "" {
			fmt.Printf("Branch: %s\n", milestone.Branch)
		}
		if milestone.Workflow != "" {
			fmt.Printf("Workflow: %s\n", milestone.Workflow)
		}
		if milestone.GithubMilestone > 0 {
			fmt.Printf("GitHub Milestone: %d\n", milestone.GithubMilestone)
		}
		if len(milestone.Assignees) > 0 {
			fmt.Printf("Assignees: %v\n", milestone.Assignees)
		}

		if milestone.Description != "" {
			fmt.Printf("\n## Description\n\n%s\n", milestone.Description)
		}

		if len(milestone.LinkedEpics) > 0 {
			fmt.Printf("\n## Linked Epics (%d)\n\n", len(milestone.LinkedEpics))
			for _, epic := range milestone.LinkedEpics {
				fmt.Printf("- %s\n", epic)
			}
		}

		if len(milestone.LinkedIssues) > 0 {
			fmt.Printf("\n## Linked Issues (%d)\n\n", len(milestone.LinkedIssues))
			for _, issue := range milestone.LinkedIssues {
				fmt.Printf("- %s  %s\n", issue.ID, issue.Title)
			}
		}

		if len(milestone.Thread) > 0 {
			fmt.Printf("\n## Thread (%d comments)\n\n", len(milestone.Thread))
			for _, comment := range milestone.Thread {
				fmt.Printf("**%s** (%s):\n%s\n\n", comment.Author, comment.Date, comment.Content)
			}
		}

		return nil
	},
}

var milestonesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new milestone",
	RunE: func(cmd *cobra.Command, args []string) error {
		if milestoneTitleFlag == "" {
			return fmt.Errorf("--title is required")
		}

		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

		req := api.CreateMilestoneRequest{
			Title:           milestoneTitleFlag,
			Description:     milestoneDescriptionFlag,
			Repo:            milestoneRepoFlag,
			Branch:          milestoneBranchFlag,
			Workflow:        milestoneWorkflowFlag,
			GithubMilestone: milestoneGithubMilestoneFlag,
			Assignees:       milestoneAssigneesFlag,
			Project:         getProject(),
		}

		milestone, err := client.CreateMilestone(req)
		if err != nil {
			return err
		}

		if jsonFlag {
			data, _ := json.MarshalIndent(milestone, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Created milestone %s: %s\n", milestone.ID, milestone.Title)
		return nil
	},
}

var milestonesUpdateCmd = &cobra.Command{
	Use:   "update [id]",
	Short: "Update an existing milestone",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

		req := api.UpdateMilestoneRequest{
			Title:           milestoneTitleFlag,
			Description:     milestoneDescriptionFlag,
			Repo:            milestoneRepoFlag,
			Branch:          milestoneBranchFlag,
			Workflow:        milestoneWorkflowFlag,
			GithubMilestone: milestoneGithubMilestoneFlag,
			Project:         getProject(),
		}

		milestone, err := client.UpdateMilestone(args[0], req)
		if err != nil {
			return err
		}

		if jsonFlag {
			data, _ := json.MarshalIndent(milestone, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Updated milestone %s: %s\n", milestone.ID, milestone.Title)
		return nil
	},
}

var milestonesEpicsCmd = &cobra.Command{
	Use:   "epics [id]",
	Short: "List epics linked to a milestone",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

		epics, err := client.ListMilestoneEpics(args[0], milestoneCompletedFlag, getProject())
		if err != nil {
			return err
		}

		if jsonFlag {
			data, _ := json.MarshalIndent(epics, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(epics) == 0 {
			fmt.Println("No epics linked to this milestone.")
			return nil
		}

		for _, epic := range epics {
			completedStr := ""
			if epic.Completed {
				completedStr = " [done]"
			}
			fmt.Printf("%s  %s%s\n", epic.ID, epic.Title, completedStr)
		}

		return nil
	},
}

var milestonesIssuesCmd = &cobra.Command{
	Use:   "issues [id]",
	Short: "List issues linked to a milestone",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

		issues, err := client.ListMilestoneIssues(args[0], milestoneCompletedFlag, getProject())
		if err != nil {
			return err
		}

		if jsonFlag {
			data, _ := json.MarshalIndent(issues, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(issues) == 0 {
			fmt.Println("No issues linked to this milestone.")
			return nil
		}

		for _, issue := range issues {
			stageStr := issue.Stage
			if issue.SubStage != "" {
				stageStr = fmt.Sprintf("%s > %s", issue.Stage, issue.SubStage)
			}
			if issue.Completed {
				stageStr += " done"
			}
			fmt.Printf("%s  %s  [%s]\n", issue.ID, issue.Title, stageStr)
		}

		return nil
	},
}

var milestonesCommentCmd = &cobra.Command{
	Use:   "comment [id]",
	Short: "Add a comment to a milestone",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if milestoneContentFlag == "" {
			return fmt.Errorf("--content is required")
		}

		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

		req := api.CommentRequest{
			Content: milestoneContentFlag,
			Persona: milestonePersonaFlag,
			Project: getProject(),
		}

		comment, err := client.AddMilestoneComment(args[0], req)
		if err != nil {
			return err
		}

		if jsonFlag {
			data, _ := json.MarshalIndent(comment, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Printf("Added comment %s to milestone %s\n", comment.ID, args[0])
		}
		return nil
	},
}

var milestonesResyncCmd = &cobra.Command{
	Use:   "resync [id]",
	Short: "Refresh linked issue titles on a milestone",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())
		result, err := client.ResyncMilestone(args[0], getProject())
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
	rootCmd.AddCommand(milestonesCmd)
	milestonesCmd.AddCommand(milestonesListCmd)
	milestonesCmd.AddCommand(milestonesGetCmd)
	milestonesCmd.AddCommand(milestonesCreateCmd)
	milestonesCmd.AddCommand(milestonesUpdateCmd)
	milestonesCmd.AddCommand(milestonesEpicsCmd)
	milestonesCmd.AddCommand(milestonesIssuesCmd)
	milestonesCmd.AddCommand(milestonesCommentCmd)
	milestonesCmd.AddCommand(milestonesResyncCmd)

	// List flags
	milestonesListCmd.Flags().BoolVar(&milestoneCompletedFlag, "completed", false, "Include completed milestones")

	// Create flags
	milestonesCreateCmd.Flags().StringVar(&milestoneTitleFlag, "title", "", "Milestone title (required)")
	milestonesCreateCmd.Flags().StringVar(&milestoneDescriptionFlag, "description", "", "Milestone description")
	milestonesCreateCmd.Flags().StringVar(&milestoneRepoFlag, "repo", "", "GitHub repository (e.g., 'eyra/mono')")
	milestonesCreateCmd.Flags().StringVar(&milestoneBranchFlag, "branch", "", "Git branch name")
	milestonesCreateCmd.Flags().StringVar(&milestoneWorkflowFlag, "workflow", "", "GitHub Actions workflow file")
	milestonesCreateCmd.Flags().IntVar(&milestoneGithubMilestoneFlag, "github-milestone", 0, "GitHub milestone number")
	milestonesCreateCmd.Flags().StringVar(&milestoneAssigneesFlag, "assignees", "", "Comma-separated list of assignee IDs")

	// Update flags
	milestonesUpdateCmd.Flags().StringVar(&milestoneTitleFlag, "title", "", "New title")
	milestonesUpdateCmd.Flags().StringVar(&milestoneDescriptionFlag, "description", "", "New description")
	milestonesUpdateCmd.Flags().StringVar(&milestoneRepoFlag, "repo", "", "GitHub repository (empty to remove)")
	milestonesUpdateCmd.Flags().StringVar(&milestoneBranchFlag, "branch", "", "Git branch name (empty to remove)")
	milestonesUpdateCmd.Flags().StringVar(&milestoneWorkflowFlag, "workflow", "", "GitHub Actions workflow file (empty to remove)")
	milestonesUpdateCmd.Flags().IntVar(&milestoneGithubMilestoneFlag, "github-milestone", 0, "GitHub milestone number (0 to remove)")

	// Epics flags
	milestonesEpicsCmd.Flags().BoolVar(&milestoneCompletedFlag, "completed", false, "Include completed epics")

	// Issues flags
	milestonesIssuesCmd.Flags().BoolVar(&milestoneCompletedFlag, "completed", false, "Include completed issues")

	// Comment flags
	milestonesCommentCmd.Flags().StringVar(&milestoneContentFlag, "content", "", "Comment content (required)")
	milestonesCommentCmd.Flags().StringVar(&milestonePersonaFlag, "persona", "", "Persona name for attribution")
}
