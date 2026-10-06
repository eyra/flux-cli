package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProtectedReads(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer personal-token" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/dev/issues/42":
			io.WriteString(w, `{"id":"42","title":"Personal issue"}`)
		case "/api/dev/personas":
			io.WriteString(w, `{"personas":[{"name":"clara"}]}`)
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

	t.Run("personas", func(t *testing.T) {
		personas, err := NewClient(server.URL, "personal-token").ListPersonas(nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(personas) != 1 || personas[0].Name != "clara" {
			t.Fatalf("unexpected personas: %+v", personas)
		}
		_, err = NewClient(server.URL, "").ListPersonas(nil)
		if !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("expected authentication error, got %v", err)
		}
	})
}

func TestGetIssueProjectAndAuthorIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/dev/issues/42" || r.URL.Query().Get("project") != "other project" {
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
	for _, comment := range output.Thread[2:] {
		if _, exists := comment["author_id"]; exists {
			t.Fatalf("invented missing author identity: %s", data)
		}
	}
}

func TestGetIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/dev/identity" {
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
