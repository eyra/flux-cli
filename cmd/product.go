package cmd

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/eyra/flux-cli/internal/api"
	"github.com/spf13/cobra"
)

// productFlags holds the flags of one Product command group (scenes or
// usecases), so the two groups never share flag values.
type productFlags struct {
	completed   bool
	noThread    bool
	scene       string
	title       string
	description string
	code        string
	area        string
	status      string
	content     string
	targetType  string
	targetID    string
}

// productCommand describes the scenes or usecases command group.
type productCommand struct {
	kind     api.ProductKind
	label    string // "scene" or "use case"
	children string // what a resync refreshes: "use cases" or "issues"
	flags    productFlags
}

var (
	scenes   = &productCommand{kind: api.KindScene, label: "scene", children: "use cases"}
	usecases = &productCommand{kind: api.KindUseCase, label: "use case", children: "issues"}
)

var scenesCmd = &cobra.Command{
	Use:   "scenes",
	Short: "Manage scenes (Product: Scene → Use Case → Issue)",
	Long: `Manage scenes in the project's Product to-do set.

A scene is an actor-centred view of the system, with one angle and one zoom
level. Use cases belong to a scene, and issues belong to a use case.
IDs may be Basecamp IDs or codes such as SCN-Next-02 (legacy UJ- codes work too).`,
}

var usecasesCmd = &cobra.Command{
	Use:   "usecases",
	Short: "Manage use cases (Product: Scene → Use Case → Issue)",
	Long: `Manage use cases in the project's Product to-do set.

A use case belongs to at most one scene, and issues belong to a use case.
IDs may be Basecamp IDs or codes such as UC-NEXT-01.`,
}

func (p *productCommand) client() *api.Client {
	return api.NewClient(baseURLForEnv(getEnv()), getAPIKey())
}

func (p *productCommand) listCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: fmt.Sprintf("List %ss", p.label),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			items, err := p.client().ListProductItems(p.kind, api.ListProductItemsOptions{
				Completed: p.flags.completed,
				Scene:     p.flags.scene,
				Project:   getProject(),
			})
			if err != nil {
				return err
			}
			if jsonFlag {
				printJSON(items)
				return nil
			}
			if len(items) == 0 {
				fmt.Printf("No %ss found.\n", p.label)
				return nil
			}
			printProductItems(items)
			return nil
		},
	}
	cmd.Flags().BoolVar(&p.flags.completed, "completed", false, fmt.Sprintf("List completed %ss", p.label))
	if p.kind == api.KindUseCase {
		cmd.Flags().StringVar(&p.flags.scene, "scene", "", "Only use cases of this scene (ID or code)")
	}
	return cmd
}

func (p *productCommand) getCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [id-or-code]",
		Short: fmt.Sprintf("Get %s details", p.label),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			item, err := p.client().GetProductItem(p.kind, args[0], getProject(), !p.flags.noThread)
			if err != nil {
				return err
			}
			if jsonFlag {
				printJSON(item)
				return nil
			}

			fmt.Printf("# %s\n\n", item.Title)
			fmt.Printf("ID: %s\n", item.ID)
			if item.Code != "" {
				fmt.Printf("Code: %s\n", item.Code)
			}
			if item.Status != "" {
				fmt.Printf("Status: %s\n", item.Status)
			}
			if item.Scene != "" {
				fmt.Printf("Scene: %s\n", item.Scene)
			}
			if item.URL != "" {
				fmt.Printf("URL: %s\n", item.URL)
			}
			if description := htmlToText(item.Description); description != "" {
				fmt.Printf("\n## Description\n\n%s\n", description)
			}
			printLinked("Linked Use Cases", item.LinkedUseCase)
			printLinked("Linked Issues", item.LinkedIssues)
			if len(item.Thread) > 0 {
				fmt.Printf("\n## Thread (%d comments)\n\n", len(item.Thread))
				for _, comment := range item.Thread {
					fmt.Printf("**%s** (%s) [%s]:\n%s\n\n", comment.Author, comment.Date, comment.ID, htmlToText(comment.ContentHTML))
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&p.flags.noThread, "no-thread", false, "Leave out the comment thread")
	return cmd
}

func (p *productCommand) createCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: fmt.Sprintf("Create a %s", p.label),
		Long: fmt.Sprintf(`Create a %s.

The title gets --code, else the code the title already starts with, else the
next free code in --area.`, p.label),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if p.flags.title == "" {
				return fmt.Errorf("--title is required")
			}
			result, err := p.client().CreateProductItem(p.kind, getProject(), api.CreateProductItemRequest{
				Title:       p.flags.title,
				Description: p.flags.description,
				Code:        p.flags.code,
				Area:        p.flags.area,
				Status:      p.flags.status,
				Scene:       p.flags.scene,
				AIModel:     getAIModel(cmd),
			})
			if err != nil {
				return err
			}
			if jsonFlag {
				printJSON(result)
				return nil
			}
			fmt.Printf("Created %s %s: %s\n", p.label, result.Value.ID, result.Value.Title)
			if result.Value.Scene != "" {
				fmt.Printf("Linked to scene %s\n", result.Value.Scene)
			}
			if result.Value.URL != "" {
				fmt.Println(result.Value.URL)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&p.flags.title, "title", "", fmt.Sprintf("%s title (required)", capitalize(p.label)))
	cmd.Flags().StringVar(&p.flags.description, "description", "", "Description")
	cmd.Flags().StringVar(&p.flags.code, "code", "", "Code, e.g. "+p.exampleCode())
	cmd.Flags().StringVar(&p.flags.area, "area", "", "Area to number the next free code in, e.g. Next")
	cmd.Flags().StringVar(&p.flags.status, "status", "", "Status: the name of a group in the list")
	if p.kind == api.KindUseCase {
		cmd.Flags().StringVar(&p.flags.scene, "scene", "", "Scene to link to (ID or code)")
	}
	cmd.Flags().String("ai-model", "", "Caller-declared model for the description footer")
	return cmd
}

func (p *productCommand) updateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update [id-or-code]",
		Short: fmt.Sprintf("Update a %s", p.label),
		Long: fmt.Sprintf(`Update a %s.

A title without a code keeps the current code; --code replaces it. A new
description replaces only the user part, not the links. --status moves the
%s to that group, or with "none" out of any group.`, p.label, p.label),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := p.client().UpdateProductItem(p.kind, args[0], getProject(), api.UpdateProductItemRequest{
				Title:       p.flags.title,
				Description: p.flags.description,
				Code:        p.flags.code,
				Status:      p.flags.status,
				AIModel:     getAIModel(cmd),
			})
			if err != nil {
				return err
			}
			if jsonFlag {
				printJSON(result)
				return nil
			}
			fmt.Printf("Updated %s %s: %s\n", p.label, result.Value.ID, result.Value.Title)
			return nil
		},
	}
	cmd.Flags().StringVar(&p.flags.title, "title", "", "New title")
	cmd.Flags().StringVar(&p.flags.description, "description", "", "New description")
	cmd.Flags().StringVar(&p.flags.code, "code", "", "New code, e.g. "+p.exampleCode())
	cmd.Flags().StringVar(&p.flags.status, "status", "", `New status: a group name, or "none"`)
	cmd.Flags().String("ai-model", "", "Caller-declared model for the supplied description footer")
	return cmd
}

func (p *productCommand) nextCodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "next-code",
		Short: fmt.Sprintf("Print the next free %s code in an area", p.label),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if p.flags.area == "" {
				return fmt.Errorf("--area is required")
			}
			result, err := p.client().NextProductCode(p.kind, p.flags.area, getProject())
			if err != nil {
				return err
			}
			if jsonFlag {
				printJSON(result)
				return nil
			}
			fmt.Println(result.Value.Code)
			return nil
		},
	}
	cmd.Flags().StringVar(&p.flags.area, "area", "", "Area, e.g. Next (required)")
	return cmd
}

func (p *productCommand) commentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment [id-or-code]",
		Short: fmt.Sprintf("Add a comment to a %s", p.label),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if p.flags.content == "" {
				return fmt.Errorf("--content is required")
			}
			result, err := p.client().AddProductComment(p.kind, args[0], getProject(), api.ProductCommentRequest{
				Content: p.flags.content,
				AIModel: getAIModel(cmd),
			})
			if err != nil {
				return err
			}
			if jsonFlag {
				printJSON(result)
				return nil
			}
			fmt.Printf("Added comment %s to %s %s\n", result.Value.ID, p.label, args[0])
			return nil
		},
	}
	cmd.Flags().StringVar(&p.flags.content, "content", "", "Comment content (required)")
	cmd.Flags().String("ai-model", "", "Caller-declared model for the content footer")
	return cmd
}

func (p *productCommand) resyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resync [id-or-code]",
		Short: fmt.Sprintf("Refresh linked %s titles on a %s", p.children, p.label),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := p.client().ResyncProductItem(p.kind, args[0], getProject())
			if err != nil {
				return err
			}
			if jsonFlag {
				printJSON(result)
				return nil
			}
			summary := result.Value
			fmt.Printf("Resynced: %d/%d linked %s updated\n", summary.Updated, summary.Checked, p.children)
			for _, update := range summary.Updates {
				fmt.Printf("- %s: %s → %s\n", update.ID, update.OldTitle, update.NewTitle)
			}
			if len(summary.NotFound) > 0 {
				ids := make([]string, len(summary.NotFound))
				for i, id := range summary.NotFound {
					ids[i] = string(id)
				}
				fmt.Printf("Not found: %s\n", strings.Join(ids, ", "))
			}
			return nil
		},
	}
}

var scenesUseCasesCmd = &cobra.Command{
	Use:   "usecases [id-or-code]",
	Short: "List the use cases linked to a scene",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := scenes.client().ListSceneUseCases(args[0], getProject())
		if err != nil {
			return err
		}
		if jsonFlag {
			printJSON(result)
			return nil
		}
		fmt.Printf("# %s\n\n", result.Value.Scene.Title)
		if len(result.Value.UseCases) == 0 {
			fmt.Println("No use cases linked to this scene.")
			return nil
		}
		printProductItems(result.Value.UseCases)
		return nil
	},
}

var usecasesIssuesCmd = &cobra.Command{
	Use:   "issues [id-or-code]",
	Short: "List the issues linked to a use case",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := usecases.client().ListUseCaseIssues(args[0], getProject())
		if err != nil {
			return err
		}
		if jsonFlag {
			printJSON(result)
			return nil
		}
		fmt.Printf("# %s\n\n", result.Value.UseCase.Title)
		if len(result.Value.Issues) == 0 {
			fmt.Println("No issues linked to this use case.")
			return nil
		}
		for _, issue := range result.Value.Issues {
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

// linkCmd builds `usecases link` or `usecases unlink`.
func (p *productCommand) linkCmd(action string) *cobra.Command {
	verb, preposition := "Link", "to"
	if action == "unlink" {
		verb, preposition = "Unlink", "from"
	}
	cmd := &cobra.Command{
		Use:   action + " [id-or-code]",
		Short: fmt.Sprintf("%s a use case %s a scene", verb, preposition),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if p.flags.targetID == "" {
				return fmt.Errorf("--target-id is required")
			}
			result, err := p.client().LinkUseCase(args[0], getProject(), api.ProductLinkRequest{
				TargetType: p.flags.targetType,
				TargetID:   p.flags.targetID,
				Action:     action,
			})
			if err != nil {
				return err
			}
			if jsonFlag {
				printJSON(result)
				return nil
			}
			fmt.Printf("%sed use case %s %s %s %s\n", verb, args[0], preposition, p.flags.targetType, p.flags.targetID)
			return nil
		},
	}
	cmd.Flags().StringVar(&p.flags.targetType, "target-type", "scene", "Target type: scene")
	cmd.Flags().StringVar(&p.flags.targetID, "target-id", "", "Scene ID or code (required)")
	return cmd
}

func (p *productCommand) exampleCode() string {
	if p.kind == api.KindUseCase {
		return "UC-NEXT-01"
	}
	return "SCN-Next-01"
}

func printProductItems(items []api.ProductItem) {
	for _, item := range items {
		line := fmt.Sprintf("%s  %s", item.ID, item.Title)
		if item.Status != "" {
			line += fmt.Sprintf("  [%s]", item.Status)
		}
		if item.UseCasesCount != nil {
			line += fmt.Sprintf("  (%s)", countLabel(*item.UseCasesCount, "use case", "use cases"))
		}
		if item.IssuesCount != nil {
			line += fmt.Sprintf("  (%s)", countLabel(*item.IssuesCount, "issue", "issues"))
		}
		fmt.Println(line)
	}
}

// countLabel returns "1 issue" or "N issues".
func countLabel(n int, singular, plural string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, singular)
	}
	return fmt.Sprintf("%d %s", n, plural)
}

func printLinked(heading string, links []api.LinkedIssue) {
	if len(links) == 0 {
		return
	}
	fmt.Printf("\n## %s (%d)\n\n", heading, len(links))
	for _, link := range links {
		fmt.Printf("- %s  %s\n", link.ID, link.Title)
	}
}

var (
	htmlBreak = regexp.MustCompile(`(?i)<br\s*/?>|</(p|div|li|h[1-6])>`)
	htmlTag   = regexp.MustCompile(`<[^>]*>`)
	blankRuns = regexp.MustCompile(`\n{3,}`)
)

// htmlToText turns Basecamp rich text into plain text for terminal output.
func htmlToText(s string) string {
	s = htmlBreak.ReplaceAllString(s, "\n")
	s = htmlTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	return strings.TrimSpace(blankRuns.ReplaceAllString(s, "\n\n"))
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func init() {
	rootCmd.AddCommand(scenesCmd, usecasesCmd)

	for _, group := range []struct {
		parent  *cobra.Command
		product *productCommand
	}{{scenesCmd, scenes}, {usecasesCmd, usecases}} {
		p := group.product
		group.parent.AddCommand(p.listCmd(), p.getCmd(), p.createCmd(), p.updateCmd(), p.commentCmd(), p.resyncCmd(), p.nextCodeCmd())
	}

	scenesCmd.AddCommand(scenesUseCasesCmd)
	usecasesCmd.AddCommand(usecases.linkCmd("link"), usecases.linkCmd("unlink"), usecasesIssuesCmd)
}
