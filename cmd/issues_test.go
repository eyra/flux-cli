package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// fixtureRequest is one request a command should send, and the answer.
type fixtureRequest struct {
	method   string
	path     string // path and query
	body     map[string]interface{}
	status   int
	response string
}

func TestIssueWriteLinks(t *testing.T) {
	const (
		created = `{"id":"3001","title":"Issue","stage":"Specification","url":"https://basecamp/3001","server_only":"kept"}`
		updated = `{"id":"3001","title":"Issue","url":"https://basecamp/3001","server_only":"kept"}`
		current = `{"id":"3001","title":"Issue","stage":"Development","epic":4001,"milestone":"5001","use_case":null,"completed":false}`
	)
	createBody := map[string]interface{}{"title": "Issue", "app": "web", "project": "fixture-project"}
	link := func(targetType, targetID, action string, status int, response string) fixtureRequest {
		return fixtureRequest{"POST", "/api/delivery/issues/3001/link",
			map[string]interface{}{"target_type": targetType, "target_id": targetID, "action": action, "project": "fixture-project"}, status, response}
	}
	for _, tc := range []struct {
		name     string
		argv     []string
		requests []fixtureRequest
		json     map[string]interface{} // fields the JSON output must have
		text     []string               // lines the text output must have
		err      string                 // error the command must return
	}{
		{
			name:     "create without links",
			argv:     []string{"issues", "create", "--title", "Issue", "--app", "web"},
			requests: []fixtureRequest{{"POST", "/api/delivery/issues", createBody, 201, created}},
			json:     map[string]interface{}{"id": "3001", "server_only": "kept"},
			text:     []string{"Created issue 3001: Issue"},
		},
		{
			name: "create with epic, milestone and use case",
			argv: []string{"issues", "create", "--title", "Issue", "--app", "web", "--epic", "4001", "--milestone", "5001", "--usecase", "UC-NEXT-01"},
			requests: []fixtureRequest{
				{"POST", "/api/delivery/issues", createBody, 201, created},
				link("epic", "4001", "link", 200, `{"action":"linked","issue_id":"3001","epic_id":"4001"}`),
				link("milestone", "5001", "link", 200, `{"action":"linked","issue_id":"3001","milestone_id":"5001"}`),
				link("use_case", "UC-NEXT-01", "link", 200, `{"action":"linked","issue_id":"3001","use_case_id":"2001","use_case_title":"UC-NEXT-01: Upload"}`),
			},
			json: map[string]interface{}{"id": "3001", "epic": "4001", "milestone": "5001", "use_case": "2001", "server_only": "kept"},
			text: []string{"Created issue 3001: Issue", "Linked to epic 4001", "Linked to milestone 5001", "Linked to use case 2001"},
		},
		{
			name: "create with a use case that fails to link",
			argv: []string{"issues", "create", "--title", "Issue", "--app", "web", "--usecase", "UC-NEXT-01"},
			requests: []fixtureRequest{
				{"POST", "/api/delivery/issues", createBody, 201, created},
				link("use_case", "UC-NEXT-01", "link", 404, `{"error":"product_not_configured","message":"This project has no Product to-do set configured"}`),
			},
			json: map[string]interface{}{"id": "3001"},
			text: []string{"Created issue 3001: Issue"},
			err:  "issue 3001 was created, but could not link use case UC-NEXT-01: product_not_configured",
		},
		{
			name:     "create with a context",
			argv:     []string{"issues", "create", "--title", "Issue", "--app", "web", "--context", "UC-NEXT-01"},
			requests: []fixtureRequest{{"POST", "/api/delivery/issues", map[string]interface{}{"title": "Issue", "app": "web", "context": "UC-NEXT-01", "project": "fixture-project"}, 201, created}},
			json:     map[string]interface{}{"id": "3001"},
			text:     []string{"Created issue 3001: Issue"},
		},
		{
			name:     "create with the deprecated program flag",
			argv:     []string{"issues", "create", "--title", "Issue", "--app", "web", "--program", "dev"},
			requests: []fixtureRequest{{"POST", "/api/delivery/issues", map[string]interface{}{"title": "Issue", "app": "web", "context": "dev", "project": "fixture-project"}, 201, created}},
			json:     map[string]interface{}{"id": "3001"},
			text:     []string{"Created issue 3001: Issue"},
		},
		{
			name:     "update the context",
			argv:     []string{"issues", "update", "3001", "--context", "Web"},
			requests: []fixtureRequest{{"PATCH", "/api/delivery/issues/3001", map[string]interface{}{"context": "Web", "project": "fixture-project"}, 200, updated}},
			json:     map[string]interface{}{"id": "3001"},
			text:     []string{"Updated issue 3001: Issue"},
		},
		{
			name: "create with an empty epic",
			argv: []string{"issues", "create", "--title", "Issue", "--app", "web", "--epic", ""},
			requests: []fixtureRequest{
				{"POST", "/api/delivery/issues", createBody, 201, created},
			},
			json: map[string]interface{}{"id": "3001"},
			text: []string{"Created issue 3001: Issue"},
		},
		{
			name: "update links an epic",
			argv: []string{"issues", "update", "3001", "--title", "Issue", "--epic", "4002"},
			requests: []fixtureRequest{
				{"PATCH", "/api/delivery/issues/3001", map[string]interface{}{"title": "Issue", "project": "fixture-project"}, 200, updated},
				link("epic", "4002", "link", 200, `{"action":"linked","issue_id":"3001","epic_id":"4002"}`),
			},
			json: map[string]interface{}{"id": "3001", "epic": "4002", "server_only": "kept"},
			text: []string{"Updated issue 3001: Issue", "Linked to epic 4002"},
		},
		{
			name: "update removes the milestone",
			argv: []string{"issues", "update", "3001", "--milestone", ""},
			requests: []fixtureRequest{
				{"PATCH", "/api/delivery/issues/3001", map[string]interface{}{"project": "fixture-project"}, 200, updated},
				{"GET", "/api/delivery/issues/3001?project=fixture-project", nil, 200, current},
				link("milestone", "5001", "unlink", 200, `{"action":"unlinked","issue_id":"3001","milestone_id":"5001"}`),
			},
			json: map[string]interface{}{"id": "3001", "milestone": nil},
			text: []string{"Updated issue 3001: Issue", "Unlinked from milestone 5001"},
		},
		{
			name: "update removes a use case the issue does not have",
			argv: []string{"issues", "update", "3001", "--usecase", ""},
			requests: []fixtureRequest{
				{"PATCH", "/api/delivery/issues/3001", map[string]interface{}{"project": "fixture-project"}, 200, updated},
				{"GET", "/api/delivery/issues/3001?project=fixture-project", nil, 200, current},
			},
			json: map[string]interface{}{"id": "3001"},
			text: []string{"Updated issue 3001: Issue"},
		},
	} {
		for _, asJSON := range []bool{true, false} {
			name := tc.name + " text"
			if asJSON {
				name = tc.name + " json"
			}
			t.Run(name, func(t *testing.T) {
				next := 0
				output, err := runCommand(t, tc.argv, asJSON, func(w http.ResponseWriter, r *http.Request) {
					if next >= len(tc.requests) {
						t.Errorf("unexpected request %s %s", r.Method, r.URL.RequestURI())
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					want := tc.requests[next]
					next++
					if r.Method != want.method || r.URL.RequestURI() != want.path {
						t.Errorf("request %d = %s %s; want %s %s", next, r.Method, r.URL.RequestURI(), want.method, want.path)
					}
					data, _ := io.ReadAll(r.Body)
					if want.body == nil {
						if len(data) != 0 {
							t.Errorf("request %d body = %s; want none", next, data)
						}
					} else {
						var body map[string]interface{}
						if err := json.Unmarshal(data, &body); err != nil || !reflect.DeepEqual(body, want.body) {
							t.Errorf("request %d body = %s; want %v", next, data, want.body)
						}
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(want.status)
					io.WriteString(w, want.response)
				})
				if tc.err == "" && err != nil {
					t.Fatal(err)
				}
				if tc.err != "" && (err == nil || !strings.Contains(err.Error(), tc.err)) {
					t.Fatalf("error = %v; want it to contain %q", err, tc.err)
				}
				if next != len(tc.requests) {
					t.Fatalf("got %d requests; want %d", next, len(tc.requests))
				}
				if asJSON {
					var got map[string]interface{}
					if err := json.Unmarshal([]byte(output), &got); err != nil {
						t.Fatalf("invalid JSON output %q: %v", output, err)
					}
					for key, value := range tc.json {
						if gotValue, ok := got[key]; !ok || !reflect.DeepEqual(gotValue, value) {
							t.Errorf("JSON %s = %v; want %v in\n%s", key, got[key], value, output)
						}
					}
					return
				}
				for _, line := range tc.text {
					if !strings.Contains(output, line+"\n") {
						t.Errorf("text output lacks %q:\n%s", line, output)
					}
				}
			})
		}
	}
}

func TestIssueContext(t *testing.T) {
	for _, tc := range []struct {
		name, response, want string
	}{
		{"context", `{"id":"1","title":"Issue","stage":"Development","context":"UC-NEXT-01","program":"UC-NEXT-01"}`, "Context: UC-NEXT-01\n"},
		{"older server", `{"id":"1","title":"Issue","stage":"Development","program":"Dev"}`, "Context: Dev\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output, err := runCommand(t, []string{"issues", "get", "1"}, false, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, tc.response)
			})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output, tc.want) {
				t.Errorf("output lacks %q:\n%s", tc.want, output)
			}
		})
	}

	t.Run("list filter", func(t *testing.T) {
		var query string
		_, err := runCommand(t, []string{"issues", "list", "--context", "UC-NEXT-01"}, true, func(w http.ResponseWriter, r *http.Request) {
			query = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"issues":[]}`)
		})
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"context=UC-NEXT-01", "program=UC-NEXT-01"} {
			if !strings.Contains(query, want) {
				t.Errorf("query %q lacks %q", query, want)
			}
		}
	})

	t.Run("program is a hidden alias", func(t *testing.T) {
		for _, command := range []*cobra.Command{issuesListCmd, issuesCreateCmd, issuesUpdateCmd} {
			flag := command.Flags().Lookup("program")
			if flag == nil || !flag.Hidden || flag.Deprecated == "" {
				t.Errorf("%s: --program should be a hidden, deprecated flag", command.Name())
			}
		}
	})
}
