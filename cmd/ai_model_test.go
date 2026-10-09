package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func isolateContentCommand(t *testing.T, command *cobra.Command) {
	t.Helper()
	isolateCommand(t)
	command.Flags().VisitAll(func(flag *pflag.Flag) {
		value, changed := flag.Value.String(), flag.Changed
		t.Cleanup(func() {
			flag.Value.Set(value)
			flag.Changed = changed
		})
		if err := flag.Value.Set(flag.DefValue); err != nil {
			t.Fatal(err)
		}
		flag.Changed = false
	})
}

func TestContentCommandsAIModelRequests(t *testing.T) {
	for _, operation := range []struct {
		name, method, path string
		command            *cobra.Command
		args               []string
		body               map[string]interface{}
	}{
		{"issue create", "POST", "/issues", issuesCreateCmd, []string{"--title", "Title", "--app", "web", "--description", "Body"}, map[string]interface{}{"title": "Title", "app": "web", "description": "Body"}},
		{"issue update", "PATCH", "/issues/42", issuesUpdateCmd, []string{"42", "--description", "Body"}, map[string]interface{}{"description": "Body"}},
		{"issue comment", "POST", "/issues/42/comments", issuesCommentCmd, []string{"42", "--content", "Body", "--persona", "sam"}, map[string]interface{}{"content": "Body"}},
		{"issue advance", "POST", "/issues/42/advance", issuesAdvanceCmd, []string{"42", "--stage", "testing", "--comment", "Body"}, map[string]interface{}{"target_stage": "testing", "comment": "Body"}},
		{"epic create", "POST", "/epics", epicsCreateCmd, []string{"--title", "Title", "--description", "Body"}, map[string]interface{}{"title": "Title", "description": "Body"}},
		{"epic update", "PATCH", "/epics/42", epicsUpdateCmd, []string{"42", "--description", "Body"}, map[string]interface{}{"description": "Body"}},
		{"epic comment", "POST", "/epics/42/comments", epicsCommentCmd, []string{"42", "--content", "Body"}, map[string]interface{}{"content": "Body"}},
		{"milestone create", "POST", "/milestones", milestonesCreateCmd, []string{"--title", "Title", "--app", "web", "--description", "Body"}, map[string]interface{}{"title": "Title", "app": "web", "description": "Body"}},
		{"milestone update", "PATCH", "/milestones/42", milestonesUpdateCmd, []string{"42", "--description", "Body"}, map[string]interface{}{"description": "Body"}},
		{"milestone comment", "POST", "/milestones/42/comments", milestonesCommentCmd, []string{"42", "--content", "Body"}, map[string]interface{}{"content": "Body"}},
		{"comment update", "PATCH", "/comments/42", commentsUpdateCmd, []string{"42", "--content", "Body", "--persona", "sam"}, map[string]interface{}{"content": "Body"}},
		{"issue metadata only", "PATCH", "/issues/42", issuesUpdateCmd, []string{"42", "--title", "Title"}, map[string]interface{}{"title": "Title"}},
		{"epic metadata only", "PATCH", "/epics/42", epicsUpdateCmd, []string{"42", "--title", "Title"}, map[string]interface{}{"title": "Title"}},
		{"milestone metadata only", "PATCH", "/milestones/42", milestonesUpdateCmd, []string{"42", "--title", "Title"}, map[string]interface{}{"title": "Title"}},
		{"advance without comment", "POST", "/issues/42/advance", issuesAdvanceCmd, []string{"42", "--stage", "testing"}, map[string]interface{}{"target_stage": "testing"}},
	} {
		t.Run(operation.name, func(t *testing.T) {
			for _, model := range []struct {
				name, value string
				supplied    bool
			}{
				{name: "omitted"},
				{name: "provider model", value: "openai/gpt-5", supplied: true},
				{name: "literal text", value: ` vendor/model <&\" @Sam> `, supplied: true},
				{name: "empty", supplied: true},
				{name: "whitespace", value: " \t\n ", supplied: true},
			} {
				t.Run(model.name, func(t *testing.T) {
					isolateContentCommand(t, operation.command)
					saveFixtureToken(t, "test")
					if err := rootCmd.PersistentFlags().Parse([]string{"--env", "test", "--project", "fixture-project", "--json"}); err != nil {
						t.Fatal(err)
					}
					want := map[string]interface{}{"project": "fixture-project"}
					for key, value := range operation.body {
						want[key] = value
					}
					args := append([]string(nil), operation.args...)
					if model.supplied {
						args = append(args, "--ai-model", model.value)
						want["ai_model"] = model.value
					}
					requests := 0
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						requests++
						if r.Method != operation.method || r.URL.Path != "/api/delivery"+operation.path || r.Header.Get("Authorization") != "Bearer fixture-key" {
							t.Errorf("unexpected request: %s %s, authorization %q", r.Method, r.URL, r.Header.Get("Authorization"))
						}
						var body map[string]interface{}
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							t.Error(err)
						} else if !reflect.DeepEqual(body, want) {
							t.Errorf("request body = %#v; want %#v", body, want)
						}
						w.Header().Set("Content-Type", "application/json")
						io.WriteString(w, `{"id":"42","title":"Title","stage_comment_id":"43","target_stage":"testing"}`)
					}))
					t.Cleanup(server.Close)
					target, err := url.Parse(server.URL)
					if err != nil {
						t.Fatal(err)
					}
					http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
						if r.URL.Host != "eyra-flux-test.fly.dev" {
							t.Errorf("unexpected environment host: %s", r.URL.Host)
						}
						request := r.Clone(r.Context())
						request.URL.Scheme = target.Scheme
						request.URL.Host = target.Host
						return server.Client().Transport.RoundTrip(request)
					})
					if err := operation.command.ParseFlags(args); err != nil {
						t.Fatal(err)
					}
					if err := operation.command.RunE(operation.command, operation.command.Flags().Args()); err != nil {
						t.Fatal(err)
					}
					if requests != 1 {
						t.Fatalf("got %d requests; want one content write", requests)
					}
				})
			}
		})
	}
}

func TestUnsupportedCommandsRejectAIModel(t *testing.T) {
	for _, command := range []*cobra.Command{
		issuesListCmd, issuesGetCmd, issuesDeleteCmd, issuesLinkCmd, issuesAssignCmd,
		epicsListCmd, epicsGetCmd, epicsIssuesCmd, epicsResyncCmd,
		milestonesListCmd, milestonesGetCmd, milestonesIssuesCmd, milestonesResyncCmd,
		commentsDeleteCmd, authStatusCmd,
	} {
		t.Run(command.CommandPath(), func(t *testing.T) {
			isolateContentCommand(t, command)
			if err := command.ParseFlags([]string{"--ai-model", "provider/model"}); err == nil || !strings.Contains(err.Error(), "unknown flag: --ai-model") {
				t.Fatalf("expected unsupported flag error, got %v", err)
			}
		})
	}
}
