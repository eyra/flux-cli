package cmd

import (
	"fmt"

	"github.com/eyra/flux-cli/internal/api"
	"github.com/spf13/cobra"
)

var commentsCmd = &cobra.Command{
	Use:   "comments",
	Short: "Manage comments",
}

var (
	commentContent string
	commentPersona string
)

var commentsUpdateCmd = &cobra.Command{
	Use:   "update [comment-id]",
	Short: "Update a comment's content",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if commentContent == "" {
			return fmt.Errorf("--content is required")
		}

		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())
		result, err := client.UpdateComment(args[0], commentContent, getProject(), commentPersona)
		if err != nil {
			return err
		}

		if jsonFlag {
			printOK("id", result.ID)
		} else {
			fmt.Printf("Comment %s updated\n", result.ID)
		}
		return nil
	},
}

var commentsDeleteCmd = &cobra.Command{
	Use:   "delete [comment-id]",
	Short: "Delete (trash) a comment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())
		result, err := client.DeleteComment(args[0], getProject())
		if err != nil {
			return err
		}

		if jsonFlag {
			printOK("id", result.ID)
		} else {
			fmt.Printf("Comment %s deleted\n", result.ID)
		}
		return nil
	},
}

func init() {
	commentsUpdateCmd.Flags().StringVar(&commentContent, "content", "", "New comment content")
	commentsUpdateCmd.Flags().StringVar(&commentPersona, "persona", "", "Persona name")

	commentsCmd.AddCommand(commentsUpdateCmd, commentsDeleteCmd)
	rootCmd.AddCommand(commentsCmd)
}
