package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
)

// captureStdout returns what fn prints to stdout.
func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = writer
	output := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, reader) //nolint:errcheck
		output <- buf.String()
	}()
	runErr := fn()
	os.Stdout = stdout
	writer.Close()
	return <-output, runErr
}

// runCommand runs `flux <argv>` against a test server that answers every
// request with handler, and returns the command's stdout.
func runCommand(t *testing.T, argv []string, json bool, handler http.HandlerFunc) (string, error) {
	t.Helper()
	command, rest, err := rootCmd.Find(argv)
	if err != nil {
		t.Fatal(err)
	}
	isolateContentCommand(t, command)
	persistent := []string{"--api-key", "fixture-key", "--env", "test", "--project", "fixture-project"}
	if json {
		persistent = append(persistent, "--json")
	}
	if err := rootCmd.PersistentFlags().Parse(persistent); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
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
	if err := command.ParseFlags(rest); err != nil {
		t.Fatal(err)
	}
	return captureStdout(t, func() error {
		return command.RunE(command, command.Flags().Args())
	})
}

func TestProductCommandRequests(t *testing.T) {
	const (
		project  = "project=fixture-project"
		item     = `{"id":"1001","code":"SCN-Next-02","title":"SCN-Next-02: Donate","name":"Donate","status":"Refine","completed":false,"comments_count":0,"url":"https://basecamp/1001","use_cases_count":1,"server_only":"kept"}`
		useCase  = `{"id":"2001","code":"UC-NEXT-01","title":"UC-NEXT-01: Upload","name":"Upload","status":null,"completed":false,"comments_count":0,"url":"https://basecamp/2001","scene":"1001","issues_count":2,"server_only":"kept"}`
		detail   = `{"id":"2001","code":"UC-NEXT-01","title":"UC-NEXT-01: Upload","status":"Refine","completed":false,"comments_count":1,"scene":"1001","issues_count":1,"project":"fixture-project","description":"<div>Upload <b>data</b></div>","thread":[{"id":"9","author":"Alex","author_id":"101","date":"2026-10-07 12:00","content_html":"<p>Hi</p>","url":"u"}],"linked_issues":[{"id":"3001","title":"Issue","url":"u"}],"server_only":"kept"}`
		write    = `{"id":"1001","code":"SCN-Next-04","title":"SCN-Next-04: Donate","url":"https://basecamp/1001","server_only":"kept"}`
		comment  = `{"id":"9","content":"Hi","url":"u","server_only":"kept"}`
		resync   = `{"parent_type":"scene","parent_id":"1001","parent_title":"SCN-Next-02: Donate","checked":3,"updated":1,"updates":[{"id":"2001","old_title":"Old","new_title":"New"}],"not_found":["2002"],"server_only":"kept"}`
		ucLink   = `{"action":"linked","use_case_id":"2001","use_case_title":"UC-NEXT-01: Upload","scene_id":"1001","scene_title":"SCN-Next-02: Donate","server_only":"kept"}`
		issueRaw = `{"id":"3001","ref":"FX-12","title":"Issue","stage":"Development","epic":4001,"milestone":"5001","use_case":"2001","url":"https://basecamp/3001","completed":false,"description_html":"<p>x</p>","server_only":"kept"}`
	)
	for _, tc := range []struct {
		name     string
		argv     []string
		method   string
		path     string // path and query
		body     map[string]interface{}
		response string
	}{
		{"scenes list", []string{"scenes", "list"}, "GET", "/api/product/scenes?" + project, nil, `{"scenes":[` + item + `]}`},
		{"scenes list completed", []string{"scenes", "list", "--completed"}, "GET", "/api/product/scenes?completed=true&" + project, nil, `{"scenes":[` + item + `]}`},
		{"scenes get", []string{"scenes", "get", "SCN-Next-02"}, "GET", "/api/product/scenes/SCN-Next-02?" + project, nil, item},
		{"scenes get without thread", []string{"scenes", "get", "1001", "--no-thread"}, "GET", "/api/product/scenes/1001?include_thread=false&" + project, nil, item},
		{"scenes create", []string{"scenes", "create", "--title", "Donate", "--area", "Next", "--description", "Body", "--status", "Refine", "--ai-model", "m"}, "POST", "/api/product/scenes?" + project,
			map[string]interface{}{"title": "Donate", "area": "Next", "description": "Body", "status": "Refine", "ai_model": "m"}, write},
		{"scenes create with code", []string{"scenes", "create", "--title", "Donate", "--code", "SCN-Next-04"}, "POST", "/api/product/scenes?" + project,
			map[string]interface{}{"title": "Donate", "code": "SCN-Next-04"}, write},
		{"scenes update", []string{"scenes", "update", "SCN-Next-02", "--title", "Donate more", "--status", "none"}, "PATCH", "/api/product/scenes/SCN-Next-02?" + project,
			map[string]interface{}{"title": "Donate more", "status": "none"}, write},
		{"scenes comment", []string{"scenes", "comment", "1001", "--content", "Hi"}, "POST", "/api/product/scenes/1001/comments?" + project,
			map[string]interface{}{"content": "Hi"}, comment},
		{"scenes usecases", []string{"scenes", "usecases", "SCN-Next-02"}, "GET", "/api/product/scenes/SCN-Next-02/use_cases?" + project, nil,
			`{"scene":{"id":"1001","code":"SCN-Next-02","title":"SCN-Next-02: Donate","url":"u"},"use_cases":[` + useCase + `],"server_only":"kept"}`},
		{"scenes resync", []string{"scenes", "resync", "SCN-Next-02"}, "POST", "/api/product/scenes/SCN-Next-02/resync?" + project, nil, resync},
		{"usecases list", []string{"usecases", "list"}, "GET", "/api/product/use_cases?" + project, nil, `{"use_cases":[` + useCase + `]}`},
		{"usecases list by scene", []string{"usecases", "list", "--scene", "SCN-Next-02", "--completed"}, "GET", "/api/product/use_cases?completed=true&" + project + "&scene=SCN-Next-02", nil, `{"use_cases":[` + useCase + `]}`},
		{"usecases get", []string{"usecases", "get", "UC-NEXT-01"}, "GET", "/api/product/use_cases/UC-NEXT-01?" + project, nil, detail},
		{"usecases create", []string{"usecases", "create", "--title", "Upload", "--area", "NEXT", "--scene", "SCN-Next-02", "--ai-model", "m"}, "POST", "/api/product/use_cases?" + project,
			map[string]interface{}{"title": "Upload", "area": "NEXT", "scene": "SCN-Next-02", "ai_model": "m"}, `{"id":"2001","code":"UC-NEXT-01","title":"UC-NEXT-01: Upload","url":"u","scene":"1001","server_only":"kept"}`},
		{"usecases update", []string{"usecases", "update", "UC-NEXT-01", "--description", "Body", "--code", "UC-NEXT-05"}, "PATCH", "/api/product/use_cases/UC-NEXT-01?" + project,
			map[string]interface{}{"description": "Body", "code": "UC-NEXT-05"}, write},
		{"usecases comment", []string{"usecases", "comment", "UC-NEXT-01", "--content", "Hi", "--ai-model", "m"}, "POST", "/api/product/use_cases/UC-NEXT-01/comments?" + project,
			map[string]interface{}{"content": "Hi", "ai_model": "m"}, comment},
		{"usecases link", []string{"usecases", "link", "UC-NEXT-01", "--target-type", "scene", "--target-id", "SCN-Next-02"}, "POST", "/api/product/use_cases/UC-NEXT-01/link?" + project,
			map[string]interface{}{"target_type": "scene", "target_id": "SCN-Next-02", "action": "link"}, ucLink},
		{"usecases link default type", []string{"usecases", "link", "2001", "--target-id", "1001"}, "POST", "/api/product/use_cases/2001/link?" + project,
			map[string]interface{}{"target_type": "scene", "target_id": "1001", "action": "link"}, ucLink},
		{"usecases unlink", []string{"usecases", "unlink", "UC-NEXT-01", "--target-type", "scene", "--target-id", "1001"}, "POST", "/api/product/use_cases/UC-NEXT-01/link?" + project,
			map[string]interface{}{"target_type": "scene", "target_id": "1001", "action": "unlink"}, strings.Replace(ucLink, `"linked"`, `"unlinked"`, 1)},
		{"usecases issues", []string{"usecases", "issues", "UC-NEXT-01"}, "GET", "/api/product/use_cases/UC-NEXT-01/issues?" + project, nil,
			`{"use_case":{"id":"2001","code":"UC-NEXT-01","title":"UC-NEXT-01: Upload","url":"u"},"issues":[` + issueRaw + `]}`},
		{"usecases resync", []string{"usecases", "resync", "UC-NEXT-01"}, "POST", "/api/product/use_cases/UC-NEXT-01/resync?" + project, nil, resync},
		{"issues link usecase", []string{"issues", "link", "3001", "--target-type", "usecase", "--target-id", "UC-NEXT-01"}, "POST", "/api/delivery/issues/3001/link",
			map[string]interface{}{"target_type": "usecase", "target_id": "UC-NEXT-01", "action": "link", "project": "fixture-project"},
			`{"action":"linked","issue_id":"3001","issue_title":"Issue","use_case_id":"2001","use_case_title":"UC-NEXT-01: Upload","server_only":"kept"}`},
		{"issues unlink usecase", []string{"issues", "link", "3001", "--target-type", "usecase", "--target-id", "2001", "--unlink"}, "POST", "/api/delivery/issues/3001/link",
			map[string]interface{}{"target_type": "usecase", "target_id": "2001", "action": "unlink", "project": "fixture-project"},
			`{"action":"unlinked","issue_id":"3001","issue_title":"Issue","use_case_id":"2001","use_case_title":"UC-NEXT-01: Upload","server_only":"kept"}`},
		{"issues get", []string{"issues", "get", "3001"}, "GET", "/api/delivery/issues/3001?" + project, nil, issueRaw},
	} {
		for _, asJSON := range []bool{true, false} {
			name := tc.name + " text"
			if asJSON {
				name = tc.name + " json"
			}
			t.Run(name, func(t *testing.T) {
				requests := 0
				output, err := runCommand(t, tc.argv, asJSON, func(w http.ResponseWriter, r *http.Request) {
					requests++
					if r.Method != tc.method || r.URL.RequestURI() != tc.path || r.Header.Get("Authorization") != "Bearer fixture-key" {
						t.Errorf("request = %s %s (authorization %q); want %s %s", r.Method, r.URL.RequestURI(), r.Header.Get("Authorization"), tc.method, tc.path)
					}
					data, _ := io.ReadAll(r.Body)
					if tc.body == nil {
						if len(data) != 0 {
							t.Errorf("request body = %s; want none", data)
						}
					} else {
						var body map[string]interface{}
						if err := json.Unmarshal(data, &body); err != nil {
							t.Errorf("request body %q: %v", data, err)
						} else if !reflect.DeepEqual(body, tc.body) {
							t.Errorf("request body = %#v; want %#v", body, tc.body)
						}
					}
					w.Header().Set("Content-Type", "application/json")
					if tc.method == "POST" && !strings.HasSuffix(r.URL.Path, "/link") && !strings.HasSuffix(r.URL.Path, "/resync") {
						w.WriteHeader(http.StatusCreated)
					}
					io.WriteString(w, tc.response)
				})
				if err != nil {
					t.Fatal(err)
				}
				if requests != 1 {
					t.Fatalf("got %d requests; want 1", requests)
				}
				if asJSON {
					if !json.Valid([]byte(output)) || !strings.Contains(output, `"server_only": "kept"`) {
						t.Fatalf("JSON output lost server fields:\n%s", output)
					}
				} else if strings.TrimSpace(output) == "" || strings.Contains(output, "server_only") {
					t.Fatalf("unexpected text output:\n%s", output)
				}
			})
		}
	}
}

func TestIssueGetJSONKeepsServerFields(t *testing.T) {
	const response = `{"id":"3001","ref":"FX-12","title":"Issue","stage":"Development","epic":4001,"milestone":"5001","use_case":null,"url":"https://basecamp/3001","completed":false,"due_on":null}`
	output, err := runCommand(t, []string{"issues", "get", "3001"}, true, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, response)
	})
	if err != nil {
		t.Fatal(err)
	}
	var got, want map[string]interface{}
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatal(err)
	}
	json.Unmarshal([]byte(response), &want) //nolint:errcheck
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("issues get --json = %v; want the server's fields %v", got, want)
	}

	output, err = runCommand(t, []string{"issues", "get", "3001"}, false, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, strings.Replace(response, `"use_case":null`, `"use_case":"2001"`, 1))
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"Ref: FX-12", "Epic: 4001", "Milestone: 5001", "Use case: 2001", "URL: https://basecamp/3001"} {
		if !strings.Contains(output, line) {
			t.Errorf("text output lacks %q:\n%s", line, output)
		}
	}
}

func TestProductCommandErrors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"product not configured", 404, `{"error":"product_not_configured","message":"This project has no Product to-do set configured"}`, "product_not_configured: this project has no Product"},
		{"older server", 404, `{"errors":{"detail":"Not Found"}}`, "not supported by this server"},
		{"older server without JSON", 404, `404 page not found`, "not supported by this server"},
		{"unknown scene", 404, `{"error":"Scene not found"}`, "not found: Scene not found"},
		{"ambiguous code", 409, `{"error":"ambiguous_code","ids":["1","2"]}`, "ambiguous_code"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runCommand(t, []string{"scenes", "get", "SCN-Next-02"}, false, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				io.WriteString(w, tc.body)
			})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v; want it to contain %q", err, tc.want)
			}
		})
	}

	t.Run("issue link without product", func(t *testing.T) {
		_, err := runCommand(t, []string{"issues", "link", "3001", "--target-type", "usecase", "--target-id", "UC-NEXT-01"}, false, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
			io.WriteString(w, `{"error":"product_not_configured","message":"This project has no Product to-do set configured"}`)
		})
		if err == nil || !strings.Contains(err.Error(), "product_not_configured") {
			t.Fatalf("error = %v; want product_not_configured", err)
		}
	})
}

func TestProductReadCommandsRejectAIModel(t *testing.T) {
	for _, argv := range [][]string{
		{"scenes", "list"}, {"scenes", "get"}, {"scenes", "usecases"}, {"scenes", "resync"},
		{"usecases", "list"}, {"usecases", "get"}, {"usecases", "issues"}, {"usecases", "resync"},
		{"usecases", "link"}, {"usecases", "unlink"},
	} {
		t.Run(strings.Join(argv, " "), func(t *testing.T) {
			command, _, err := rootCmd.Find(argv)
			if err != nil {
				t.Fatal(err)
			}
			isolateContentCommand(t, command)
			if err := command.ParseFlags([]string{"--ai-model", "provider/model"}); err == nil || !strings.Contains(err.Error(), "unknown flag: --ai-model") {
				t.Fatalf("expected unsupported flag error, got %v", err)
			}
		})
	}
}
