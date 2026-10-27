package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestProtectedReads(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer personal-token" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/delivery/issues/42":
			io.WriteString(w, `{"id":"42","title":"Personal issue"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	t.Run("issue", func(t *testing.T) {
		issue, err := NewClient(server.URL, "personal-token").GetIssue("42", "flux")
		if err != nil {
			t.Fatal(err)
		}
		if issue.ID != "42" || issue.Title != "Personal issue" {
			t.Fatalf("unexpected issue: %+v", issue)
		}
		_, err = NewClient(server.URL, "").GetIssue("42", "flux")
		if !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("expected authentication error, got %v", err)
		}
	})
}

func TestGetIssueProjectAndAuthorIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/delivery/issues/42" || r.URL.Query().Get("project") != "other project" {
			http.NotFound(w, r)
			return
		}
		io.WriteString(w, `{"id":"42","thread":[
			{"id":"1","author":"Alex","author_id":"101"},
			{"id":"2","author":"Alex","author_id":"202"},
			{"id":"3","author":"Alex","author_id":null},
			{"id":"4","author":"Alex"}
		]}`)
	}))
	t.Cleanup(server.Close)

	issue, err := NewClient(server.URL, "personal-token").GetIssue("42", "other project")
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(issue)
	if err != nil {
		t.Fatal(err)
	}
	var output struct {
		Thread []map[string]interface{} `json:"thread"`
	}
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatal(err)
	}
	if len(output.Thread) != 4 {
		t.Fatalf("unexpected thread: %s", data)
	}
	for i, id := range []string{"101", "202"} {
		if output.Thread[i]["author"] != "Alex" || output.Thread[i]["author_id"] != id {
			t.Fatalf("lost distinct author identity: %s", data)
		}
	}
	// JSON output repeats the server's JSON: a null author stays null and a
	// missing one stays missing.
	if id, exists := output.Thread[2]["author_id"]; !exists || id != nil {
		t.Fatalf("changed null author identity: %s", data)
	}
	if _, exists := output.Thread[3]["author_id"]; exists {
		t.Fatalf("invented missing author identity: %s", data)
	}
}

func TestGetIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/delivery/identity" {
			http.NotFound(w, r)
			return
		}
		switch r.Header.Get("Authorization") {
		case "Bearer personal-token":
			io.WriteString(w, `{"basecamp_account_id":"99","basecamp_person_id":"101","display_name":"Alex"}`)
		case "Bearer incomplete":
			io.WriteString(w, `{"display_name":"Alex"}`)
		default:
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
	}))
	t.Cleanup(server.Close)

	identity, err := NewClient(server.URL, "personal-token").GetIdentity()
	if err != nil {
		t.Fatal(err)
	}
	if *identity != (Identity{BasecampAccountID: "99", BasecampPersonID: "101", DisplayName: "Alex"}) {
		t.Fatalf("unexpected identity: %+v", identity)
	}
	for _, key := range []string{"", "invalid"} {
		identity, err := NewClient(server.URL, key).GetIdentity()
		if identity != nil || !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("expected unauthorized for %q, got %+v, %v", key, identity, err)
		}
	}
	identity, err = NewClient(server.URL, "incomplete").GetIdentity()
	if identity != nil || err == nil {
		t.Fatalf("accepted incomplete identity: %+v, %v", identity, err)
	}
}

func TestDeliveryFallsBackToLegacyPrefix(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		switch r.URL.Path {
		case "/api/dev/issues/42":
			io.WriteString(w, `{"id":"42","title":"Old server"}`)
		case "/api/dev/issues/42/link":
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"target_type":"epic"`) {
				t.Errorf("retried link lost its body: %s", body)
			}
			io.WriteString(w, `{"action":"linked"}`)
		case "/api/dev/issues/404":
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, `{"error":"Issue not found"}`)
		default:
			// An older Phoenix server has no /api/delivery routes.
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, `{"errors":{"detail":"Not Found"}}`)
		}
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "key")
	issue, err := client.GetIssue("42", "")
	if err != nil || issue.Title != "Old server" {
		t.Fatalf("GetIssue = %+v, %v", issue, err)
	}
	if _, err := client.LinkIssue("42", LinkRequest{TargetType: "epic", TargetID: "7"}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetIssue("404", ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
	want := []string{
		"GET /api/delivery/issues/42", "GET /api/dev/issues/42",
		"POST /api/dev/issues/42/link",
		"GET /api/dev/issues/404",
	}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("requests = %v; want %v", paths, want)
	}
}

func TestDeliveryNotFoundDoesNotFallBack(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, `{"error":"Issue not found"}`)
	}))
	t.Cleanup(server.Close)

	_, err := NewClient(server.URL, "key").GetIssue("42", "")
	if !errors.Is(err, ErrNotFound) || !strings.Contains(err.Error(), "Issue not found") {
		t.Fatalf("expected issue not found, got %v", err)
	}
	if !reflect.DeepEqual(paths, []string{"/api/delivery/issues/42"}) {
		t.Fatalf("requests = %v; want only /api/delivery", paths)
	}
}

func TestUploadImageFallsBackToLegacyPrefix(t *testing.T) {
	file := filepath.Join(t.TempDir(), "image.png")
	if err := os.WriteFile(file, []byte("png"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/dev/images" {
			http.NotFound(w, r)
			return
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil || r.MultipartForm.File["file"] == nil {
			t.Errorf("retried upload lost its form: %v", err)
		}
		io.WriteString(w, `{"sgid":"s","html":"h"}`)
	}))
	t.Cleanup(server.Close)

	result, err := NewClient(server.URL, "key").UploadImage(file, "", "")
	if err != nil || result.SGID != "s" {
		t.Fatalf("UploadImage = %+v, %v", result, err)
	}
}

func TestNotifications(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/delivery/notifications":
			if r.URL.RawQuery != "mentions=true&page=2&project=flux&read=true" {
				t.Errorf("unexpected query %q", r.URL.RawQuery)
			}
			io.WriteString(w, `{"notifications":[{"id":1,"type":"Mention","flux_id":"42","readable_sgid":"sg1"}]}`)
		case "/api/delivery/notifications/read":
			body, _ := io.ReadAll(r.Body)
			if r.Method != http.MethodPost || string(body) != `{"readables":["sg1"]}` {
				t.Errorf("unexpected %s body %s", r.Method, body)
			}
			io.WriteString(w, `{"ok":true}`)
		}
	}))
	t.Cleanup(server.Close)
	client := NewClient(server.URL, "token")

	list, err := client.ListNotifications("flux", true, 2, true)
	if err != nil || len(list) != 1 || *list[0].FluxID != "42" || list[0].ReadableSGID != "sg1" {
		t.Fatalf("unexpected %+v, %v", list, err)
	}
	if err := client.MarkNotificationsRead([]string{"sg1"}); err != nil {
		t.Fatalf("unexpected %v", err)
	}
}

// Incident listings as eyra-flux-test sends them (2026-10-27): AppSignal's
// Markdown wrapped per type, trimmed.
const (
	exceptionListing = `{"type":"exception","content":"# Exception Incidents\nShowing **1** of **1** exception incidents (1-1)\n\n## Incident #121 - ArgumentError\n├─ **State**: OPEN\n└─ **Preview**: ` + "```** (ArgumentError) unknown status code 520```" + `\n"}`
	anomalyListing   = `{"type":"anomaly","content":"# Anomaly Detection Incidents\n**No anomaly incidents found** matching your criteria.\n"}`
)

func TestListIncidents(t *testing.T) {
	for _, tc := range []struct {
		name, kind, response string
		want                 []IncidentList
	}{
		{"all types", "", `{"incidents":{"anomalies":` + anomalyListing + `,"exceptions":` + exceptionListing + `}}`,
			[]IncidentList{{Type: "exception"}, {Type: "anomaly"}}},
		{"exceptions", "exception", `{"incidents":` + exceptionListing + `}`, []IncidentList{{Type: "exception"}}},
		{"anomalies", "anomaly", `{"incidents":` + anomalyListing + `}`, []IncidentList{{Type: "anomaly"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/delivery/appsignal/incidents" || r.URL.Query().Get("app") != "Flux/prod" || r.URL.Query().Get("type") != tc.kind {
					http.NotFound(w, r)
					return
				}
				io.WriteString(w, tc.response)
			}))
			t.Cleanup(server.Close)

			lists, err := NewClient(server.URL, "token").ListIncidents(ListIncidentsOptions{App: "Flux/prod", Type: tc.kind})
			if err != nil {
				t.Fatal(err)
			}
			if len(lists) != len(tc.want) {
				t.Fatalf("got %d listings, want %d: %+v", len(lists), len(tc.want), lists)
			}
			for i, list := range lists {
				if list.Type != tc.want[i].Type {
					t.Errorf("listing %d: type %q, want %q", i, list.Type, tc.want[i].Type)
				}
				if !strings.HasPrefix(list.Content, "# ") {
					t.Errorf("listing %d: content not passed through: %q", i, list.Content)
				}
			}
			if len(lists) > 0 && lists[0].Type == "exception" && !strings.Contains(lists[0].Content, "## Incident #121 - ArgumentError") {
				t.Errorf("exception content lost: %q", lists[0].Content)
			}
		})
	}
}

func TestListIncidentsUnexpectedShape(t *testing.T) {
	// None of these may read as "no incidents".
	for _, tc := range []struct{ name, response string }{
		{"null", `{"incidents":null}`},
		{"empty object", `{"incidents":{}}`},
		{"missing key", `{}`},
		{"listing without type", `{"incidents":{"content":"# Exception Incidents\n"}}`},
		{"type listing without type", `{"incidents":{"exceptions":{"content":"# Exception Incidents\n"}}}`},
		{"other object", `{"incidents":{"items":[]}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				io.WriteString(w, tc.response)
			}))
			t.Cleanup(server.Close)

			lists, err := NewClient(server.URL, "token").ListIncidents(ListIncidentsOptions{App: "Flux/prod"})
			if !errors.Is(err, ErrUnexpectedIncidents) {
				t.Fatalf("expected unexpected incidents error, got %v (listings %+v)", err, lists)
			}
		})
	}
}

// An incident and an app's resources as eyra-flux-test sends them
// (2026-10-27), trimmed.
const (
	incidentResponse  = `{"number":121,"content":"# Incident #121: ArgumentError\n├─ **Type**: ArgumentError\n└─ **Assigned**: ⚠️ Unassigned\n"}`
	resourcesResponse = `{"sections":["namespaces"],"content":"# Application Resources\nRequested sections: namespaces\n\n## Namespaces (1)\n└─ **web**\n"}`
)

func TestGetIncident(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/delivery/appsignal/incidents/121" || r.URL.Query().Get("app") != "Flux/prod" {
			http.NotFound(w, r)
			return
		}
		io.WriteString(w, incidentResponse)
	}))
	t.Cleanup(server.Close)

	incident, err := NewClient(server.URL, "token").GetIncident("Flux/prod", 121)
	if err != nil {
		t.Fatal(err)
	}
	if incident.Number != 121 || !strings.HasPrefix(incident.Content, "# Incident #121: ArgumentError\n") {
		t.Fatalf("unexpected incident: %+v", incident)
	}
}

func TestGetAppSignalResources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/delivery/appsignal/resources" || r.URL.Query().Get("app") != "Flux/prod" || r.URL.Query().Get("sections") != "namespaces" {
			http.NotFound(w, r)
			return
		}
		io.WriteString(w, resourcesResponse)
	}))
	t.Cleanup(server.Close)

	resources, err := NewClient(server.URL, "token").GetAppSignalResources("Flux/prod", "namespaces")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(resources.Sections, []string{"namespaces"}) || !strings.Contains(resources.Content, "## Namespaces (1)\n└─ **web**") {
		t.Fatalf("unexpected resources: %+v", resources)
	}
}

func TestAppSignalUnexpectedShapes(t *testing.T) {
	// None of these may print as an empty incident or empty resources.
	for _, tc := range []struct {
		name, response string
		call           func(*Client) error
		want           error
	}{
		{"incident null", `null`, getIncident, ErrUnexpectedIncident},
		{"incident empty", `{}`, getIncident, ErrUnexpectedIncident},
		{"incident without content", `{"number":121}`, getIncident, ErrUnexpectedIncident},
		{"incident without number", `{"content":"# Incident #121: ArgumentError\n"}`, getIncident, ErrUnexpectedIncident},
		{"incident structured", `{"number":121,"name":"ArgumentError","state":"open"}`, getIncident, ErrUnexpectedIncident},
		{"resources null", `null`, getResources, ErrUnexpectedResources},
		{"resources empty", `{}`, getResources, ErrUnexpectedResources},
		{"resources without content", `{"sections":["namespaces"]}`, getResources, ErrUnexpectedResources},
		{"resources without sections", `{"content":"# Application Resources\n"}`, getResources, ErrUnexpectedResources},
		{"resources structured", `{"users":[{"id":"1","name":"Alex"}],"namespaces":["web"]}`, getResources, ErrUnexpectedResources},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				io.WriteString(w, tc.response)
			}))
			t.Cleanup(server.Close)

			if err := tc.call(NewClient(server.URL, "token")); !errors.Is(err, tc.want) {
				t.Fatalf("expected %v, got %v", tc.want, err)
			}
		})
	}
}

func getIncident(c *Client) error {
	_, err := c.GetIncident("Flux/prod", 121)
	return err
}

func getResources(c *Client) error {
	_, err := c.GetAppSignalResources("Flux/prod", "")
	return err
}
