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
