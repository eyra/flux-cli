package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"testing"
)

func TestAppSignalIncidentsList(t *testing.T) {
	// The all-types response as eyra-flux-test sends it (2026-10-27), trimmed.
	const response = `{"incidents":{
		"anomalies":{"type":"anomaly","content":"# Anomaly Detection Incidents\n**No anomaly incidents found** matching your criteria.\n"},
		"exceptions":{"type":"exception","content":"# Exception Incidents\n## Incident #121 - ArgumentError\n├─ **State**: OPEN\n"}}}`
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/delivery/appsignal/incidents" || r.URL.Query().Get("app") != "Flux/prod" {
			t.Errorf("unexpected request: %s", r.URL)
		}
		io.WriteString(w, response)
	}
	argv := []string{"appsignal", "incidents", "list", "--app", "Flux/prod"}

	t.Run("text", func(t *testing.T) {
		output, err := runCommand(t, argv, false, handler)
		if err != nil {
			t.Fatal(err)
		}
		want := "# Exception Incidents\n## Incident #121 - ArgumentError\n├─ **State**: OPEN\n\n" +
			"# Anomaly Detection Incidents\n**No anomaly incidents found** matching your criteria.\n"
		if output != want {
			t.Fatalf("output:\n%s\nwant:\n%s", output, want)
		}
	})

	t.Run("json", func(t *testing.T) {
		output, err := runCommand(t, argv, true, handler)
		if err != nil {
			t.Fatal(err)
		}
		var got []map[string]string
		if err := json.Unmarshal([]byte(output), &got); err != nil {
			t.Fatalf("output is not a JSON list: %v\n%s", err, output)
		}
		want := []map[string]string{
			{"type": "exception", "content": "# Exception Incidents\n## Incident #121 - ArgumentError\n├─ **State**: OPEN\n"},
			{"type": "anomaly", "content": "# Anomaly Detection Incidents\n**No anomaly incidents found** matching your criteria.\n"},
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})
}
