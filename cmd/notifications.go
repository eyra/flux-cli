package cmd

import (
	"fmt"

	"github.com/eyra/flux-cli/internal/api"
	"github.com/spf13/cobra"
)

var notificationsCmd = &cobra.Command{
	Use:   "notifications",
	Short: "Show your Basecamp notifications for the project",
	Long: `Show your unread Basecamp notifications for the project.

Without --json the listed notifications are marked as read.
With --json they are only marked as read with --mark-read.
--read shows already-read notifications and never marks anything.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mentions, _ := cmd.Flags().GetBool("mentions")
		read, _ := cmd.Flags().GetBool("read")
		page, _ := cmd.Flags().GetInt("page")
		markRead, _ := cmd.Flags().GetBool("mark-read")

		client := api.NewClient(baseURLForEnv(getEnv()), getAccessToken())
		notifications, err := client.ListNotifications(getProject(), read, page, mentions)
		if err != nil {
			return err
		}

		if jsonFlag {
			printJSON(notifications)
		} else {
			if len(notifications) == 0 {
				if read {
					fmt.Printf("No read notifications on page %d\n", page)
				} else {
					fmt.Println("No unread notifications")
				}
				return nil
			}
			for _, n := range notifications {
				fmt.Printf("[%s] %s\n", n.Type, n.Title)
				if n.Creator != nil {
					fmt.Printf("  by %s", *n.Creator)
				}
				fmt.Printf("  at %s\n", n.CreatedAt)
				if n.FluxID != nil {
					fmt.Printf("  Flux ID: %s\n", *n.FluxID)
				}
				fmt.Printf("  %s\n\n", n.URL)
			}
			markRead = true
		}

		if read || !markRead || len(notifications) == 0 {
			return nil
		}
		readables := make([]string, len(notifications))
		for i, n := range notifications {
			readables[i] = n.ReadableSGID
		}
		marked, err := client.MarkNotificationsRead(readables)
		if err != nil {
			return err
		}
		if !jsonFlag {
			fmt.Printf("Marked %d as read\n", marked)
		}
		return nil
	},
}

func init() {
	notificationsCmd.Flags().Bool("mentions", false, "Only show mentions")
	notificationsCmd.Flags().Bool("read", false, "Show already-read notifications instead (never marks anything)")
	notificationsCmd.Flags().Int("page", 1, "Page of read notifications, 50 per page (with --read)")
	notificationsCmd.Flags().Bool("mark-read", false, "With --json, mark the listed notifications as read")
	rootCmd.AddCommand(notificationsCmd)
}
