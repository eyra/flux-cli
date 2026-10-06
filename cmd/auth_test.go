package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/eyra/flux-cli/internal/api"
	"github.com/eyra/flux-cli/internal/auth"
	"github.com/spf13/cobra"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func isolateCommand(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("FLUX_ENV", "")
	t.Setenv("FLUX_API_KEY", "")
	for _, name := range []string{"env", "api-key", "project", "json"} {
		flag := rootCmd.PersistentFlags().Lookup(name)
		value, changed := flag.Value.String(), flag.Changed
		t.Cleanup(func() {
			flag.Value.Set(value)
			flag.Changed = changed
		})
		if err := flag.Value.Set(flag.DefValue); err != nil {
			t.Fatal(err)
		}
		flag.Changed = false
	}
	transport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = transport })
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		return nil, errors.New("unexpected request")
	})
}

func TestAuthStatusVerifiedIdentity(t *testing.T) {
	for _, source := range []string{"personal", "environment key", "explicit key"} {
		t.Run(source, func(t *testing.T) {
			isolateCommand(t)
			t.Setenv("FLUX_ENV", "test")
			if err := rootCmd.PersistentFlags().Parse([]string{"--env", "prod", "--json"}); err != nil {
				t.Fatal(err)
			}
			if err := auth.Save("prod", &auth.Credentials{
				AccessToken: "prod-personal", RefreshToken: "private-refresh", ExpiresAt: time.Now().Add(time.Hour),
			}); err != nil {
				t.Fatal(err)
			}
			if err := auth.Save("test", &auth.Credentials{
				AccessToken: "wrong-environment", ExpiresAt: time.Now().Add(time.Hour),
			}); err != nil {
				t.Fatal(err)
			}
			credential := "prod-personal"
			if source != "personal" {
				t.Setenv("FLUX_API_KEY", "environment-key")
				credential = "environment-key"
			}
			if source == "explicit key" {
				if err := rootCmd.PersistentFlags().Set("api-key", "explicit-key"); err != nil {
					t.Fatal(err)
				}
				credential = "explicit-key"
			}
			requests := 0
			http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				requests++
				if r.Method != http.MethodGet || r.URL.String() != "https://eyra-flux.fly.dev/api/dev/identity" || r.Header.Get("Authorization") != "Bearer "+credential {
					t.Errorf("wrong environment, endpoint, or credential selection")
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"basecamp_account_id":"99","basecamp_person_id":"101","display_name":"Alex","access_token":"must-not-leak"}`)), Header: make(http.Header)}, nil
			})
			var output bytes.Buffer
			command := &cobra.Command{}
			command.SetOut(&output)
			if err := authStatusCmd.RunE(command, nil); err != nil {
				t.Fatal(err)
			}
			var status map[string]interface{}
			if err := json.Unmarshal(output.Bytes(), &status); err != nil {
				t.Fatal(err)
			}
			want := map[string]interface{}{
				"signed_in": true, "env": "prod", "basecamp_account_id": "99", "basecamp_person_id": "101", "display_name": "Alex",
			}
			if requests != 1 || !reflect.DeepEqual(status, want) {
				t.Fatalf("unexpected verified status (%d requests): %s", requests, output.String())
			}
		})
	}
}

func TestAuthStatusFailsClosed(t *testing.T) {
	for _, scenario := range []string{"missing", "rejected saved token", "incomplete identity", "invalid JSON", "unavailable"} {
		t.Run(scenario, func(t *testing.T) {
			isolateCommand(t)
			jsonFlag = true
			if scenario != "missing" {
				if err := auth.Save("prod", &auth.Credentials{AccessToken: "saved-token", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
					t.Fatal(err)
				}
				http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
					status, body := http.StatusUnauthorized, `{"error":"unauthorized"}`
					switch scenario {
					case "incomplete identity":
						status, body = http.StatusOK, `{"display_name":"Alex"}`
					case "invalid JSON":
						status, body = http.StatusOK, `not JSON`
					case "unavailable":
						return nil, errors.New("connection refused")
					}
					return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
				})
			}
			var output bytes.Buffer
			command := &cobra.Command{}
			command.SetOut(&output)
			err := authStatusCmd.RunE(command, nil)
			if err == nil || output.Len() != 0 {
				t.Fatalf("expected failure without success output, got %v, %q", err, output.String())
			}
			if (scenario == "missing" || scenario == "rejected saved token") && !errors.Is(err, api.ErrUnauthorized) {
				t.Fatalf("expected authentication error, got %v", err)
			}
		})
	}
}

func TestEnvironmentAndProjectSelection(t *testing.T) {
	for _, tc := range []struct {
		name, environment, wantEnv, wantProject string
		args                                    []string
	}{
		{name: "default", wantEnv: "prod", wantProject: "next"},
		{name: "environment", environment: "test", wantEnv: "test", wantProject: "flux"},
		{name: "explicit production", environment: "test", args: []string{"--env", "prod"}, wantEnv: "prod", wantProject: "next"},
		{name: "explicit test", environment: "prod", args: []string{"--env", "test"}, wantEnv: "test", wantProject: "flux"},
		{name: "explicit project", environment: "test", args: []string{"--project", "next"}, wantEnv: "test", wantProject: "next"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolateCommand(t)
			t.Setenv("FLUX_ENV", tc.environment)
			if err := rootCmd.PersistentFlags().Parse(tc.args); err != nil {
				t.Fatal(err)
			}
			if getEnv() != tc.wantEnv || getProject() != tc.wantProject {
				t.Fatalf("got env/project %s/%s; want %s/%s", getEnv(), getProject(), tc.wantEnv, tc.wantProject)
			}
		})
	}
}
