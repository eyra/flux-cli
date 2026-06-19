package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/eyra/flux-cli/internal/api"
	"github.com/spf13/cobra"
)

var diagramsCmd = &cobra.Command{
	Use:   "diagrams",
	Short: "Render Mermaid diagrams and upload to Basecamp",
}

var (
	diagramFileFlag    string
	diagramMermaidFlag string
	diagramCaptionFlag string
)

var diagramsRenderCmd = &cobra.Command{
	Use:   "render",
	Short: "Render a Mermaid diagram, returns sgid and HTML snippet",
	RunE: func(cmd *cobra.Command, args []string) error {
		mermaid := diagramMermaidFlag
		if mermaid == "" && diagramFileFlag != "" {
			data, err := os.ReadFile(diagramFileFlag)
			if err != nil {
				return fmt.Errorf("cannot read file: %w", err)
			}
			mermaid = string(data)
		}
		if mermaid == "" {
			return fmt.Errorf("--file or --mermaid is required")
		}

		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())
		result, err := client.RenderDiagram(mermaid, diagramCaptionFlag, getProject())
		if err != nil {
			return err
		}

		if jsonFlag {
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Printf("SGID: %s\n\nHTML snippet:\n%s\n", result.SGID, result.HTML)
		}
		return nil
	},
}

func init() {
	diagramsRenderCmd.Flags().StringVar(&diagramFileFlag, "file", "", "Path to .mmd file")
	diagramsRenderCmd.Flags().StringVar(&diagramMermaidFlag, "mermaid", "", "Mermaid markup (alternative to --file)")
	diagramsRenderCmd.Flags().StringVar(&diagramCaptionFlag, "caption", "Diagram", "Caption for the diagram")

	diagramsCmd.AddCommand(diagramsRenderCmd)
	rootCmd.AddCommand(diagramsCmd)
}
