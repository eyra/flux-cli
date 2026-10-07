package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/eyra/flux-cli/internal/api"
	"github.com/spf13/cobra"
)

var (
	stageFlag          string
	issueCompletedFlag bool
	// Create flags
	issueTitleFlag       string
	issueDescriptionFlag string
	issueContextFlag     string
	issueSizeFlag        string
	issuePriorityFlag    int
	issueAppFlag         string
	issueEpicFlag        string
	issueMilestoneFlag   string
	issueUseCaseFlag     string
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
			Context:   issueContextFlag,
			App:       issueAppFlag,
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
			if issue.Completed {
				stageStr += " done"
			}
			if issue.App != "" {
				fmt.Printf("%s  %s  [%s]  %s\n", issue.ID, issue.Title, stageStr, issue.App)
			} else {
				fmt.Printf("%s  %s  [%s]\n", issue.ID, issue.Title, stageStr)
			}
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

		issue, err := client.GetIssue(args[0], getProject())
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
		if issue.Ref != "" {
			fmt.Printf("Ref: %s\n", issue.Ref)
		}
		fmt.Printf("Stage: %s", issue.Stage)
		if issue.SubStage != "" {
			fmt.Printf(" > %s", issue.SubStage)
		}
		fmt.Println()
		if issue.Completed {
			fmt.Printf("Completed: yes\n")
		}

		if context := issue.DisplayContext(); context != "" {
			fmt.Printf("Context: %s\n", context)
		}
		if issue.Size != "" {
			fmt.Printf("Size: %s\n", issue.Size)
		}
		if issue.App != "" {
			fmt.Printf("App: %s\n", issue.App)
		}
		if issue.Epic != "" {
			fmt.Printf("Epic: %s\n", issue.Epic)
		}
		if issue.Milestone != "" {
			fmt.Printf("Milestone: %s\n", issue.Milestone)
		}
		if issue.UseCase != "" {
			fmt.Printf("Use case: %s\n", issue.UseCase)
		}
		if issue.URL != "" {
			fmt.Printf("URL: %s\n", issue.URL)
		}
		if len(issue.Assignees) > 0 {
			names := make([]string, len(issue.Assignees))
			for i, p := range issue.Assignees {
				names[i] = p.Name
			}
			fmt.Printf("Assignees: %s\n", strings.Join(names, ", "))
		}

		fmt.Printf("\n## Description\n\n%s\n", issue.Description)

		if len(issue.Thread) > 0 {
			fmt.Printf("\n## Thread (%d comments)\n\n", len(issue.Thread))
			for _, comment := range issue.Thread {
				fmt.Printf("**%s** (%s) [%s]:\n%s\n\n", comment.Author, comment.Date, comment.ID, comment.Content)
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
		if issueAppFlag == "" {
			return fmt.Errorf("--app is required (web, ios, or android)")
		}

		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

		req := api.CreateIssueRequest{
			Title:       issueTitleFlag,
			Description: issueDescriptionFlag,
			Stage:       stageFlag,
			Context:     issueContextFlag,
			Size:        issueSizeFlag,
			Priority:    issuePriorityFlag,
			App:         issueAppFlag,
			Persona:     issuePersonaFlag,
			Project:     getProject(),
			AIModel:     getAIModel(cmd),
		}

		issue, err := client.CreateIssue(req)
		if err != nil {
			return err
		}

		links, linkErr := linkIssueParents(cmd, client, issue.ID, false)
		printIssueWrite("Created", issue, links)
		if linkErr != nil {
			return fmt.Errorf("issue %s was created, but %w", issue.ID, linkErr)
		}
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
			Context:     issueContextFlag,
			Description: issueDescriptionFlag,
			Size:        issueSizeFlag,
			Priority:    issuePriorityFlag,
			App:         issueAppFlag,
			Persona:     issuePersonaFlag,
			Project:     getProject(),
			AIModel:     getAIModel(cmd),
		}

		issue, err := client.UpdateIssue(args[0], req)
		if err != nil {
			return err
		}

		links, linkErr := linkIssueParents(cmd, client, args[0], true)
		printIssueWrite("Updated", issue, links)
		if linkErr != nil {
			return fmt.Errorf("issue %s was updated, but %w", args[0], linkErr)
		}
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
			AIModel: getAIModel(cmd),
		}

		comment, err := client.AddIssueComment(args[0], req)
		if err != nil {
			return err
		}

		if jsonFlag {
			data, _ := json.MarshalIndent(comment, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Printf("Added comment %s to issue %s\n", comment.ID, args[0])
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
			AIModel:        getAIModel(cmd),
		}

		result, err := client.AdvanceIssue(args[0], req)
		if err != nil {
			return err
		}

		if jsonFlag {
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
		} else {
			stage := result.TargetStage
			if stage == "" {
				stage = issueTargetStageFlag
			}
			if stage != "" {
				fmt.Printf("Advanced issue %s to %s\n", args[0], stage)
			} else {
				fmt.Printf("Advanced issue %s\n", args[0])
			}
			fmt.Printf("Stage comment: %s\n", result.StageCommentID)
			if result.UserCommentID != "" {
				fmt.Printf("Comment: %s\n", result.UserCommentID)
			}
		}
		return nil
	},
}

var issuesLinkCmd = &cobra.Command{
	Use:   "link [id]",
	Short: "Link an issue to an epic, milestone or use case",
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

		result, err := client.LinkIssue(args[0], req)
		if err != nil {
			return err
		}

		if jsonFlag {
			printServerOK(result, "id", args[0])
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

// issueParents are the parents that issues create and update link to through
// a flag. The server ignores them in the issue body, so the CLI links each one
// through the issue's link endpoint after the write.
var issueParents = []struct{ flag, targetType, label string }{
	{"epic", "epic", "epic"},
	{"milestone", "milestone", "milestone"},
	{"usecase", "use_case", "use case"},
}

// issueLink is a link that issues create or update made or removed.
type issueLink struct {
	targetType string
	label      string
	id         string
	unlinked   bool
}

// linkIssueParents links issue id to the parents given by flags. With
// allowUnlink, an empty flag value removes the issue's current link of that
// type. It returns the links it made before any error.
func linkIssueParents(cmd *cobra.Command, client *api.Client, id string, allowUnlink bool) ([]issueLink, error) {
	var links []issueLink
	var current *api.Issue
	for _, parent := range issueParents {
		if !cmd.Flags().Changed(parent.flag) {
			continue
		}
		target, _ := cmd.Flags().GetString(parent.flag)
		action := "link"
		if target == "" {
			if !allowUnlink {
				continue
			}
			if current == nil {
				issue, err := client.GetIssue(id, getProject())
				if err != nil {
					return links, fmt.Errorf("could not read its links: %w", err)
				}
				current = issue
			}
			if target = currentParent(current, parent.targetType); target == "" {
				continue
			}
			action = "unlink"
		}

		response, err := client.LinkIssue(id, api.LinkRequest{
			TargetType: parent.targetType,
			TargetID:   target,
			Action:     action,
			Project:    getProject(),
		})
		if err != nil {
			return links, fmt.Errorf("could not %s %s %s: %w", action, parent.label, target, err)
		}
		links = append(links, issueLink{
			targetType: parent.targetType,
			label:      parent.label,
			id:         linkedParentID(response, parent.targetType, target),
			unlinked:   action == "unlink",
		})
	}
	return links, nil
}

func currentParent(issue *api.Issue, targetType string) string {
	switch targetType {
	case "epic":
		return string(issue.Epic)
	case "milestone":
		return string(issue.Milestone)
	default:
		return string(issue.UseCase)
	}
}

// linkedParentID returns the parent ID from a link response (a use case code
// resolves to its ID there), or fallback.
func linkedParentID(response json.RawMessage, targetType, fallback string) string {
	var fields map[string]interface{}
	json.Unmarshal(response, &fields) //nolint:errcheck
	switch id := fields[targetType+"_id"].(type) {
	case string:
		if id != "" {
			return id
		}
	case float64:
		return fmt.Sprintf("%.0f", id)
	}
	return fallback
}

// printIssueWrite prints a created or updated issue with the links made.
// JSON output is the server's issue with the link fields set.
func printIssueWrite(verb string, issue *api.Issue, links []issueLink) {
	if jsonFlag {
		data, _ := json.Marshal(issue)
		var fields map[string]interface{}
		decoder := json.NewDecoder(strings.NewReader(string(data)))
		decoder.UseNumber()
		if err := decoder.Decode(&fields); err != nil || fields == nil {
			fields = map[string]interface{}{}
		}
		for _, link := range links {
			if link.unlinked {
				fields[link.targetType] = nil
			} else {
				fields[link.targetType] = link.id
			}
		}
		printJSON(fields)
		return
	}

	fmt.Printf("%s issue %s: %s\n", verb, issue.ID, issue.Title)
	for _, link := range links {
		if link.unlinked {
			fmt.Printf("Unlinked from %s %s\n", link.label, link.id)
		} else {
			fmt.Printf("Linked to %s %s\n", link.label, link.id)
		}
	}
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
	issuesListCmd.Flags().StringVar(&issueAppFlag, "app", "", "Filter by app (web, ios, android)")
	addContextFlags(issuesListCmd, "Filter by context, the bracketed title prefix (e.g. dev, UC-NEXT-01)")

	// Create flags
	issuesCreateCmd.Flags().StringVar(&issueTitleFlag, "title", "", "Issue title (required)")
	issuesCreateCmd.Flags().StringVar(&issueDescriptionFlag, "description", "", "Issue description")
	issuesCreateCmd.Flags().StringVarP(&stageFlag, "stage", "s", "", "Target stage (specification, design, development, testing)")
	addContextFlags(issuesCreateCmd, "Context, set as the bracketed title prefix (e.g. Dev, UC-NEXT-01)")
	issuesCreateCmd.Flags().StringVar(&issueSizeFlag, "size", "", "Size estimate (S, M, L, XL)")
	issuesCreateCmd.Flags().IntVar(&issuePriorityFlag, "priority", 0, "Priority 1-4 for tech debt (1 = highest)")
	issuesCreateCmd.Flags().StringVar(&issueAppFlag, "app", "", "App (web, ios, android)")
	issuesCreateCmd.Flags().StringVar(&issueEpicFlag, "epic", "", "Epic ID to link to")
	issuesCreateCmd.Flags().StringVar(&issueMilestoneFlag, "milestone", "", "Milestone ID to link to")
	issuesCreateCmd.Flags().StringVar(&issueUseCaseFlag, "usecase", "", "Use case ID or code to link to, e.g. UC-NEXT-01")
	issuesCreateCmd.Flags().StringVar(&issuePersonaFlag, "persona", "", "Persona name for attribution")
	issuesCreateCmd.Flags().String("ai-model", "", "Caller-declared model for the description footer")

	// Update flags
	issuesUpdateCmd.Flags().StringVar(&issueTitleFlag, "title", "", "New title")
	addContextFlags(issuesUpdateCmd, "Context, replaces the bracketed title prefix (e.g. Dev, UC-NEXT-01)")
	issuesUpdateCmd.Flags().StringVar(&issueDescriptionFlag, "description", "", "New description")
	issuesUpdateCmd.Flags().StringVar(&issueSizeFlag, "size", "", "Size estimate (S, M, L, XL)")
	issuesUpdateCmd.Flags().IntVar(&issuePriorityFlag, "priority", 0, "Priority 1-4 for tech debt (1 = highest)")
	issuesUpdateCmd.Flags().StringVar(&issueAppFlag, "app", "", "App (web, ios, android)")
	issuesUpdateCmd.Flags().StringVar(&issueEpicFlag, "epic", "", "Epic ID to link to (empty to remove)")
	issuesUpdateCmd.Flags().StringVar(&issueMilestoneFlag, "milestone", "", "Milestone ID to link to (empty to remove)")
	issuesUpdateCmd.Flags().StringVar(&issueUseCaseFlag, "usecase", "", "Use case ID or code to link to (empty to remove)")
	issuesUpdateCmd.Flags().StringVar(&issuePersonaFlag, "persona", "", "Persona name for attribution")
	issuesUpdateCmd.Flags().String("ai-model", "", "Caller-declared model for the supplied description footer")

	// Comment flags
	issuesCommentCmd.Flags().StringVar(&issueCommentContentFlag, "content", "", "Comment content (required)")
	issuesCommentCmd.Flags().StringVar(&issuePersonaFlag, "persona", "", "Persona name for attribution")
	issuesCommentCmd.Flags().String("ai-model", "", "Caller-declared model for the content footer")

	// Advance flags
	issuesAdvanceCmd.Flags().StringVar(&issueTargetStageFlag, "stage", "", "Target stage (specification, design, development, testing)")
	issuesAdvanceCmd.Flags().StringVar(&issueTargetSubstageFlag, "substage", "", "Target sub-stage")
	issuesAdvanceCmd.Flags().StringVar(&issueAdvanceCommentFlag, "comment", "", "Comment explaining the transition")
	issuesAdvanceCmd.Flags().StringVar(&issuePersonaFlag, "persona", "", "Persona name for attribution")
	issuesAdvanceCmd.Flags().String("ai-model", "", "Caller-declared model for the optional comment footer")

	// Link flags
	issuesLinkCmd.Flags().StringVar(&issueTargetTypeFlag, "target-type", "", "Target type: epic, milestone or usecase (required)")
	issuesLinkCmd.Flags().StringVar(&issueTargetIDFlag, "target-id", "", "Target ID or, for a use case, its code (required)")
	issuesLinkCmd.Flags().BoolVar(&issueUnlinkFlag, "unlink", false, "Unlink instead of link")

	// Assign flags
	issuesAssignCmd.Flags().StringVar(&issueAssigneeIDsFlag, "assignees", "", "Comma-separated person IDs (required)")

	issuesCmd.AddCommand(issuesAssignCmd)
}

// addContextFlags adds --context, and --program as its hidden, deprecated
// alias: "program" was the old name of an issue's context.
func addContextFlags(cmd *cobra.Command, usage string) {
	cmd.Flags().StringVar(&issueContextFlag, "context", "", usage)
	cmd.Flags().StringVar(&issueContextFlag, "program", "", usage)
	cmd.Flags().MarkDeprecated("program", "use --context instead")
}
