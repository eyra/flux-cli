package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/eyra/flux-cli/internal/api"
	"github.com/spf13/cobra"
)

var (
	// AppSignal flags
	appsignalAppFlag       string
	appsignalTypeFlag      string
	appsignalStateFlag     string
	appsignalNamespaceFlag string
	appsignalStartFlag     string
	appsignalEndFlag       string
	appsignalSeverityFlag  string
	appsignalAssignFlag    string
	appsignalUnassignFlag  string
	appsignalSectionsFlag  string
	appsignalContentFlag   string
)

var appsignalCmd = &cobra.Command{
	Use:   "appsignal",
	Short: "Manage AppSignal incidents and resources",
}

var appsignalAppsCmd = &cobra.Command{
	Use:   "apps",
	Short: "List available AppSignal applications",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

		apps, err := client.ListAppSignalApps()
		if err != nil {
			return err
		}

		if jsonFlag {
			data, _ := json.MarshalIndent(apps, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(apps) == 0 {
			fmt.Println("No AppSignal apps found.")
			return nil
		}

		for _, app := range apps {
			fmt.Println(app)
		}

		return nil
	},
}

var appsignalIncidentsCmd = &cobra.Command{
	Use:   "incidents",
	Short: "Manage AppSignal incidents",
}

var appsignalIncidentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List incidents",
	RunE: func(cmd *cobra.Command, args []string) error {
		if appsignalAppFlag == "" {
			return fmt.Errorf("--app is required")
		}

		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

		opts := api.ListIncidentsOptions{
			App:       appsignalAppFlag,
			Type:      appsignalTypeFlag,
			State:     appsignalStateFlag,
			Namespace: appsignalNamespaceFlag,
			Start:     appsignalStartFlag,
			End:       appsignalEndFlag,
		}

		incidents, err := client.ListIncidents(opts)
		if err != nil {
			return err
		}

		if jsonFlag {
			data, _ := json.MarshalIndent(incidents, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(incidents) == 0 {
			fmt.Println("No incidents found.")
			return nil
		}

		for _, incident := range incidents {
			severityStr := ""
			if incident.Severity != "" {
				severityStr = fmt.Sprintf(" [%s]", incident.Severity)
			}
			fmt.Printf("#%d  %s  (%s)%s\n", incident.Number, incident.Name, incident.State, severityStr)
		}

		return nil
	},
}

var appsignalIncidentsGetCmd = &cobra.Command{
	Use:   "get [number]",
	Short: "Get incident details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if appsignalAppFlag == "" {
			return fmt.Errorf("--app is required")
		}

		number, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid incident number: %s", args[0])
		}

		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

		incident, err := client.GetIncident(appsignalAppFlag, number)
		if err != nil {
			return err
		}

		if jsonFlag {
			data, _ := json.MarshalIndent(incident, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("# Incident #%d: %s\n\n", incident.Number, incident.Name)
		fmt.Printf("State: %s\n", incident.State)
		if incident.Severity != "" {
			fmt.Printf("Severity: %s\n", incident.Severity)
		}
		if incident.Type != "" {
			fmt.Printf("Type: %s\n", incident.Type)
		}
		if incident.Namespace != "" {
			fmt.Printf("Namespace: %s\n", incident.Namespace)
		}
		if incident.Occurrences > 0 {
			fmt.Printf("Occurrences: %d\n", incident.Occurrences)
		}
		if len(incident.Assignees) > 0 {
			fmt.Printf("Assignees: %v\n", incident.Assignees)
		}

		if incident.Message != "" {
			fmt.Printf("\n## Message\n\n%s\n", incident.Message)
		}

		if incident.StackTrace != "" {
			fmt.Printf("\n## Stack Trace\n\n%s\n", incident.StackTrace)
		}

		if len(incident.Notes) > 0 {
			fmt.Printf("\n## Notes (%d)\n\n", len(incident.Notes))
			for _, note := range incident.Notes {
				fmt.Printf("**%s** (%s):\n%s\n\n", note.Author, note.Date, note.Content)
			}
		}

		return nil
	},
}

var appsignalIncidentsUpdateCmd = &cobra.Command{
	Use:   "update [numbers]",
	Short: "Update incident(s) state, severity, or assignment",
	Long:  "Update one or more incidents. Numbers can be comma-separated (e.g., '123' or '123,456,789').",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if appsignalAppFlag == "" {
			return fmt.Errorf("--app is required")
		}

		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

		req := api.UpdateIncidentRequest{
			State:    appsignalStateFlag,
			Severity: appsignalSeverityFlag,
			Assign:   appsignalAssignFlag,
			Unassign: appsignalUnassignFlag,
		}

		if err := client.UpdateIncidents(appsignalAppFlag, args[0], req); err != nil {
			return err
		}

		fmt.Printf("Updated incident(s) %s\n", args[0])
		return nil
	},
}

var appsignalIncidentsNoteCmd = &cobra.Command{
	Use:   "note [number]",
	Short: "Add a note to an incident",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if appsignalAppFlag == "" {
			return fmt.Errorf("--app is required")
		}
		if appsignalContentFlag == "" {
			return fmt.Errorf("--content is required")
		}

		number, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid incident number: %s", args[0])
		}

		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

		if err := client.AddIncidentNote(appsignalAppFlag, number, appsignalContentFlag); err != nil {
			return err
		}

		fmt.Printf("Added note to incident #%d\n", number)
		return nil
	},
}

var appsignalResourcesCmd = &cobra.Command{
	Use:   "resources",
	Short: "Get available resources for an AppSignal app",
	Long:  "Returns users (for assignment), notifiers, namespaces, and dashboards.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if appsignalAppFlag == "" {
			return fmt.Errorf("--app is required")
		}

		client := api.NewClient(baseURLForEnv(getEnv()), getAPIKey())

		resources, err := client.GetAppSignalResources(appsignalAppFlag, appsignalSectionsFlag)
		if err != nil {
			return err
		}

		if jsonFlag {
			data, _ := json.MarshalIndent(resources, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(resources.Users) > 0 {
			fmt.Println("## Users")
			for _, user := range resources.Users {
				fmt.Printf("  %s - %s\n", user.ID, user.Name)
			}
			fmt.Println()
		}

		if len(resources.Notifiers) > 0 {
			fmt.Println("## Notifiers")
			for _, notifier := range resources.Notifiers {
				fmt.Printf("  %s - %s (%s)\n", notifier.ID, notifier.Name, notifier.Type)
			}
			fmt.Println()
		}

		if len(resources.Namespaces) > 0 {
			fmt.Println("## Namespaces")
			for _, ns := range resources.Namespaces {
				fmt.Printf("  %s\n", ns)
			}
			fmt.Println()
		}

		if len(resources.Dashboards) > 0 {
			fmt.Println("## Dashboards")
			for _, dashboard := range resources.Dashboards {
				fmt.Printf("  %s - %s\n", dashboard.ID, dashboard.Name)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(appsignalCmd)
	appsignalCmd.AddCommand(appsignalAppsCmd)
	appsignalCmd.AddCommand(appsignalIncidentsCmd)
	appsignalCmd.AddCommand(appsignalResourcesCmd)

	appsignalIncidentsCmd.AddCommand(appsignalIncidentsListCmd)
	appsignalIncidentsCmd.AddCommand(appsignalIncidentsGetCmd)
	appsignalIncidentsCmd.AddCommand(appsignalIncidentsUpdateCmd)
	appsignalIncidentsCmd.AddCommand(appsignalIncidentsNoteCmd)

	// Common app flag
	appsignalIncidentsListCmd.Flags().StringVar(&appsignalAppFlag, "app", "", "AppSignal app (e.g., 'MyApp/production') (required)")
	appsignalIncidentsGetCmd.Flags().StringVar(&appsignalAppFlag, "app", "", "AppSignal app (e.g., 'MyApp/production') (required)")
	appsignalIncidentsUpdateCmd.Flags().StringVar(&appsignalAppFlag, "app", "", "AppSignal app (e.g., 'MyApp/production') (required)")
	appsignalIncidentsNoteCmd.Flags().StringVar(&appsignalAppFlag, "app", "", "AppSignal app (e.g., 'MyApp/production') (required)")
	appsignalResourcesCmd.Flags().StringVar(&appsignalAppFlag, "app", "", "AppSignal app (e.g., 'MyApp/production') (required)")

	// List flags
	appsignalIncidentsListCmd.Flags().StringVar(&appsignalTypeFlag, "type", "", "Incident type: exception, anomaly, or all")
	appsignalIncidentsListCmd.Flags().StringVar(&appsignalStateFlag, "state", "", "Filter by state (open, closed, wip for exceptions; open, closed, warmup, cooldown, archived for anomalies)")
	appsignalIncidentsListCmd.Flags().StringVar(&appsignalNamespaceFlag, "namespace", "", "Filter by namespace (e.g., 'web', 'background')")
	appsignalIncidentsListCmd.Flags().StringVar(&appsignalStartFlag, "start", "", "Start of time range (ISO 8601)")
	appsignalIncidentsListCmd.Flags().StringVar(&appsignalEndFlag, "end", "", "End of time range (ISO 8601)")

	// Update flags
	appsignalIncidentsUpdateCmd.Flags().StringVar(&appsignalStateFlag, "state", "", "New state: open, closed, or wip")
	appsignalIncidentsUpdateCmd.Flags().StringVar(&appsignalSeverityFlag, "severity", "", "New severity: critical, high, low, none, informational, or untriaged")
	appsignalIncidentsUpdateCmd.Flags().StringVar(&appsignalAssignFlag, "assign", "", "Comma-separated user IDs to assign")
	appsignalIncidentsUpdateCmd.Flags().StringVar(&appsignalUnassignFlag, "unassign", "", "Comma-separated user IDs to unassign")

	// Note flags
	appsignalIncidentsNoteCmd.Flags().StringVar(&appsignalContentFlag, "content", "", "Note content (required)")

	// Resources flags
	appsignalResourcesCmd.Flags().StringVar(&appsignalSectionsFlag, "sections", "", "Comma-separated sections: users, notifiers, namespaces, dashboards")
}
