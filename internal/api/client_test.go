package api

import (
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
		issue, err := NewClient(server.URL, "personal-token").GetIssue("42")
		if err != nil {
			t.Fatal(err)
		}
		if issue.ID != "42" || issue.Title != "Personal issue" {
			t.Fatalf("unexpected issue: %+v", issue)
		}
		_, err = NewClient(server.URL, "").GetIssue("42")
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
