package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestPersonasCommandIsRemoved(t *testing.T) {
	for _, command := range rootCmd.Commands() {
		if command.Name() == "personas" {
			t.Fatal("personas command still exists")
		}
	}
}

func TestPersonaFlagIsAHiddenNoOp(t *testing.T) {
	for _, command := range []*cobra.Command{
		issuesCreateCmd, issuesUpdateCmd, issuesCommentCmd, issuesAdvanceCmd,
		epicsCommentCmd, milestonesCommentCmd, commentsUpdateCmd,
	} {
		t.Run(command.CommandPath(), func(t *testing.T) {
			flag := command.Flags().Lookup("persona")
			if flag == nil {
				t.Fatal("--persona must still parse, so existing scripts keep working")
			}
			if !flag.Hidden || flag.Deprecated == "" {
				t.Fatalf("--persona must be hidden and deprecated, got hidden=%v deprecated=%q", flag.Hidden, flag.Deprecated)
			}
		})
	}
}
