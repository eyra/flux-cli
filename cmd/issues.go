package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/eyra/flux-cli/internal/api"
	"github.com/spf13/cobra"
)

var (
	stageFlag          string
	issueCompletedFlag bool
	// Create flags
	issueTitleFlag       string
	issueDescriptionFlag string
	issueProgramFlag     string
	issueSizeFlag        string
	issuePriorityFlag    int
	issueEpicFlag        string
	issueMilestoneFlag   string
	issuePersonaFlag     string
	// Advance flags
	issueTargetStageFlag    string
	issueTargetSubstageFlag string
	issueAdvanceCommentFlag string
	// Link flags
	issueTargetTypeFlag string
	issueTargetIDFlag   string
	issueUnlinkFlag     bool
	// Comment flag
	issueCommentContentFlag string
	// Assign flag
	issueAssigneeIDsFlag string
)

var issuesCmd = &cobra.Command{
	Use:   "issues",
	Short: "Manage issues",
}

var issuesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List issues",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

		issues, err := client.ListIssues(api.ListIssuesOptions{
				Stage:     stageFlag,
				Completed: issueCompletedFlag,
				Project:   getProject(),
			})
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
		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

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

var issuesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new issue",
	RunE: func(cmd *cobra.Command, args []string) error {
		if issueTitleFlag == "" {
			return fmt.Errorf("--title is required")
		}

		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

		req := api.CreateIssueRequest{
			Title:       issueTitleFlag,
			Description: issueDescriptionFlag,
			Stage:       stageFlag,
			Program:     issueProgramFlag,
			Size:        issueSizeFlag,
			Priority:    issuePriorityFlag,
			Epic:        issueEpicFlag,
			Milestone:   issueMilestoneFlag,
			Persona:     issuePersonaFlag,
			Project:     getProject(),
		}

		issue, err := client.CreateIssue(req)
		if err != nil {
			return err
		}

		if jsonFlag {
			data, _ := json.MarshalIndent(issue, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Created issue %s: %s\n", issue.ID, issue.Title)
		return nil
	},
}

var issuesUpdateCmd = &cobra.Command{
	Use:   "update [id]",
	Short: "Update an existing issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

		req := api.UpdateIssueRequest{
			Title:       issueTitleFlag,
			Description: issueDescriptionFlag,
			Size:        issueSizeFlag,
			Priority:    issuePriorityFlag,
			Epic:        issueEpicFlag,
			Milestone:   issueMilestoneFlag,
			Persona:     issuePersonaFlag,
			Project:     getProject(),
		}

		issue, err := client.UpdateIssue(args[0], req)
		if err != nil {
			return err
		}

		if jsonFlag {
			data, _ := json.MarshalIndent(issue, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Updated issue %s: %s\n", issue.ID, issue.Title)
		return nil
	},
}

var issuesDeleteCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

		if err := client.DeleteIssue(args[0], getProject()); err != nil {
			return err
		}

		if jsonFlag {
			printOK("id", args[0])
		} else {
			fmt.Printf("Deleted issue %s\n", args[0])
		}
		return nil
	},
}

var issuesCommentCmd = &cobra.Command{
	Use:   "comment [id]",
	Short: "Add a comment to an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if issueCommentContentFlag == "" {
			return fmt.Errorf("--content is required")
		}

		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

		req := api.CommentRequest{
			Content: issueCommentContentFlag,
			Persona: issuePersonaFlag,
			Project: getProject(),
		}

		if err := client.AddIssueComment(args[0], req); err != nil {
			return err
		}

		if jsonFlag {
			printOK("id", args[0])
		} else {
			fmt.Printf("Added comment to issue %s\n", args[0])
		}
		return nil
	},
}

var issuesAdvanceCmd = &cobra.Command{
	Use:   "advance [id]",
	Short: "Advance an issue to the next stage",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

		req := api.AdvanceIssueRequest{
			TargetStage:    issueTargetStageFlag,
			TargetSubstage: issueTargetSubstageFlag,
			Comment:        issueAdvanceCommentFlag,
			Persona:        issuePersonaFlag,
			Project:        getProject(),
		}

		if err := client.AdvanceIssue(args[0], req); err != nil {
			return err
		}

		if jsonFlag {
			printOK("id", args[0], "stage", issueTargetStageFlag)
		} else if issueTargetStageFlag != "" {
			fmt.Printf("Advanced issue %s to %s\n", args[0], issueTargetStageFlag)
		} else {
			fmt.Printf("Advanced issue %s\n", args[0])
		}
		return nil
	},
}

var issuesLinkCmd = &cobra.Command{
	Use:   "link [id]",
	Short: "Link an issue to an epic or milestone",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if issueTargetTypeFlag == "" || issueTargetIDFlag == "" {
			return fmt.Errorf("--target-type and --target-id are required")
		}

		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

		action := "link"
		if issueUnlinkFlag {
			action = "unlink"
		}

		req := api.LinkRequest{
			TargetType: issueTargetTypeFlag,
			TargetID:   issueTargetIDFlag,
			Action:     action,
			Project:    getProject(),
		}

		if err := client.LinkIssue(args[0], req); err != nil {
			return err
		}

		if jsonFlag {
			printOK("id", args[0])
		} else if issueUnlinkFlag {
			fmt.Printf("Unlinked issue %s from %s %s\n", args[0], issueTargetTypeFlag, issueTargetIDFlag)
		} else {
			fmt.Printf("Linked issue %s to %s %s\n", args[0], issueTargetTypeFlag, issueTargetIDFlag)
		}
		return nil
	},
}

var issuesAssignCmd = &cobra.Command{
	Use:   "assign [id]",
	Short: "Assign people to an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if issueAssigneeIDsFlag == "" {
			return fmt.Errorf("--assignees is required")
		}
		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())
		req := api.AssignIssueRequest{
			AssigneeIDs: issueAssigneeIDsFlag,
			Project:     getProject(),
		}
		if err := client.AssignIssue(args[0], req); err != nil {
			return err
		}
		if jsonFlag {
			printOK("id", args[0])
		} else {
			fmt.Printf("Issue %s assigned\n", args[0])
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(issuesCmd)
	issuesCmd.AddCommand(issuesListCmd)
	issuesCmd.AddCommand(issuesGetCmd)
	issuesCmd.AddCommand(issuesCreateCmd)
	issuesCmd.AddCommand(issuesUpdateCmd)
	issuesCmd.AddCommand(issuesDeleteCmd)
	issuesCmd.AddCommand(issuesCommentCmd)
	issuesCmd.AddCommand(issuesAdvanceCmd)
	issuesCmd.AddCommand(issuesLinkCmd)

	// List flags
	issuesListCmd.Flags().StringVarP(&stageFlag, "stage", "s", "", "Filter by stage (specification, design, development, testing)")
	issuesListCmd.Flags().BoolVar(&issueCompletedFlag, "completed", false, "List only completed issues")

	// Create flags
	issuesCreateCmd.Flags().StringVar(&issueTitleFlag, "title", "", "Issue title (required)")
	issuesCreateCmd.Flags().StringVar(&issueDescriptionFlag, "description", "", "Issue description")
	issuesCreateCmd.Flags().StringVarP(&stageFlag, "stage", "s", "", "Target stage (specification, design, development, testing)")
	issuesCreateCmd.Flags().StringVar(&issueProgramFlag, "program", "", "Program (basecamp, github, make, mcp, dev, devops)")
	issuesCreateCmd.Flags().StringVar(&issueSizeFlag, "size", "", "Size estimate (S, M, L, XL)")
	issuesCreateCmd.Flags().IntVar(&issuePriorityFlag, "priority", 0, "Priority 1-4 for tech debt (1 = highest)")
	issuesCreateCmd.Flags().StringVar(&issueEpicFlag, "epic", "", "Epic ID to link to")
	issuesCreateCmd.Flags().StringVar(&issueMilestoneFlag, "milestone", "", "Milestone ID to link to")
	issuesCreateCmd.Flags().StringVar(&issuePersonaFlag, "persona", "", "Persona name for attribution")

	// Update flags
	issuesUpdateCmd.Flags().StringVar(&issueTitleFlag, "title", "", "New title")
	issuesUpdateCmd.Flags().StringVar(&issueDescriptionFlag, "description", "", "New description")
	issuesUpdateCmd.Flags().StringVar(&issueSizeFlag, "size", "", "Size estimate (S, M, L, XL)")
	issuesUpdateCmd.Flags().IntVar(&issuePriorityFlag, "priority", 0, "Priority 1-4 for tech debt (1 = highest)")
	issuesUpdateCmd.Flags().StringVar(&issueEpicFlag, "epic", "", "Epic ID to link to (empty to remove)")
	issuesUpdateCmd.Flags().StringVar(&issueMilestoneFlag, "milestone", "", "Milestone ID to link to (empty to remove)")
	issuesUpdateCmd.Flags().StringVar(&issuePersonaFlag, "persona", "", "Persona name for attribution")

	// Comment flags
	issuesCommentCmd.Flags().StringVar(&issueCommentContentFlag, "content", "", "Comment content (required)")
	issuesCommentCmd.Flags().StringVar(&issuePersonaFlag, "persona", "", "Persona name for attribution")

	// Advance flags
	issuesAdvanceCmd.Flags().StringVar(&issueTargetStageFlag, "stage", "", "Target stage (specification, design, development, testing)")
	issuesAdvanceCmd.Flags().StringVar(&issueTargetSubstageFlag, "substage", "", "Target sub-stage")
	issuesAdvanceCmd.Flags().StringVar(&issueAdvanceCommentFlag, "comment", "", "Comment explaining the transition")
	issuesAdvanceCmd.Flags().StringVar(&issuePersonaFlag, "persona", "", "Persona name for attribution")

	// Link flags
	issuesLinkCmd.Flags().StringVar(&issueTargetTypeFlag, "target-type", "", "Target type: epic or milestone (required)")
	issuesLinkCmd.Flags().StringVar(&issueTargetIDFlag, "target-id", "", "Target ID (required)")
	issuesLinkCmd.Flags().BoolVar(&issueUnlinkFlag, "unlink", false, "Unlink instead of link")

	// Assign flags
	issuesAssignCmd.Flags().StringVar(&issueAssigneeIDsFlag, "assignees", "", "Comma-separated person IDs (required)")

	issuesCmd.AddCommand(issuesAssignCmd)
}
