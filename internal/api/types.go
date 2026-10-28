package api

import (
	"bytes"
	"encoding/json"
	"errors"
)

// Identity is the server-verified Basecamp principal for the active credentials.
type Identity struct {
	BasecampAccountID string `json:"basecamp_account_id"`
	BasecampPersonID  string `json:"basecamp_person_id"`
	DisplayName       string `json:"display_name"`
}

// Project represents a Flux project
type Project struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// ProjectsResponse is the response from listing projects
type ProjectsResponse struct {
	Projects []Project `json:"projects"`
}

// CreateIssueRequest is the request body for creating an issue. Links to an
// epic, milestone or use case go through LinkIssue afterwards: the server
// ignores them here.
type CreateIssueRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	Stage       string  `json:"stage,omitempty"`
	Context     string  `json:"context,omitempty"`
	Size        string  `json:"size,omitempty"`
	Priority    int     `json:"priority,omitempty"`
	App         string  `json:"app,omitempty"`
	Project     string  `json:"project,omitempty"`
	AIModel     *string `json:"ai_model,omitempty"`
}

// UpdateIssueRequest is the request body for updating an issue. Links go
// through LinkIssue, as for CreateIssueRequest.
type UpdateIssueRequest struct {
	Title       string  `json:"title,omitempty"`
	Context     string  `json:"context,omitempty"`
	Description string  `json:"description,omitempty"`
	Size        string  `json:"size,omitempty"`
	Priority    int     `json:"priority,omitempty"`
	App         string  `json:"app,omitempty"`
	Project     string  `json:"project,omitempty"`
	AIModel     *string `json:"ai_model,omitempty"`
}

// AdvanceIssueRequest is the request body for advancing an issue stage
type AdvanceIssueRequest struct {
	TargetStage    string  `json:"target_stage,omitempty"`
	TargetSubstage string  `json:"target_substage,omitempty"`
	Comment        string  `json:"comment,omitempty"`
	Project        string  `json:"project,omitempty"`
	AIModel        *string `json:"ai_model,omitempty"`
}

// LinkRequest is the request body for linking/unlinking items
type LinkRequest struct {
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	Action     string `json:"action,omitempty"` // "link" or "unlink"
	Project    string `json:"project,omitempty"`
}

// CommentRequest is the request body for adding or updating a comment
type CommentRequest struct {
	Content string  `json:"content"`
	Project string  `json:"project,omitempty"`
	AIModel *string `json:"ai_model,omitempty"`
}

// CommentResponse is returned when a comment is created
type CommentResponse struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// LinkedIssue represents an issue linked to an epic or milestone
type LinkedIssue struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

// Epic represents an epic in the backlog
type Epic struct {
	ID           string        `json:"id"`
	Title        string        `json:"title"`
	Description  string        `json:"description,omitempty"`
	Milestone    string        `json:"milestone,omitempty"`
	MilestoneID  string        `json:"milestone_id,omitempty"`
	Branch       string        `json:"branch,omitempty"`
	Assignees    []string      `json:"assignees,omitempty"`
	Completed    bool          `json:"completed"`
	LinkedIssues []LinkedIssue `json:"linked_issues,omitempty"`
	Thread       []Comment     `json:"thread,omitempty"`
}

// EpicsResponse is the response from listing epics
type EpicsResponse struct {
	Epics []Epic `json:"epics"`
}

// CreateEpicRequest is the request body for creating an epic
type CreateEpicRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	Milestone   string  `json:"milestone,omitempty"`
	Assignees   string  `json:"assignee_ids,omitempty"` // comma-separated IDs
	Project     string  `json:"project,omitempty"`
	AIModel     *string `json:"ai_model,omitempty"`
}

// UpdateEpicRequest is the request body for updating an epic
type UpdateEpicRequest struct {
	Title       string  `json:"title,omitempty"`
	Description string  `json:"description,omitempty"`
	Milestone   string  `json:"milestone,omitempty"`
	Branch      string  `json:"branch,omitempty"`
	Project     string  `json:"project,omitempty"`
	AIModel     *string `json:"ai_model,omitempty"`
}

// Milestone represents a milestone in release planning
type Milestone struct {
	ID              string        `json:"id"`
	Title           string        `json:"title"`
	Description     string        `json:"description,omitempty"`
	App             string        `json:"app,omitempty"`
	Repo            string        `json:"repo,omitempty"`
	Branch          string        `json:"branch,omitempty"`
	Workflow        string        `json:"workflow,omitempty"`
	GithubMilestone int           `json:"github_milestone,omitempty"`
	Assignees       []Person      `json:"assignees,omitempty"`
	Completed       bool          `json:"completed"`
	LinkedEpics     []string      `json:"linked_epics,omitempty"`
	LinkedIssues    []LinkedIssue `json:"linked_issues,omitempty"`
	Thread          []Comment     `json:"thread,omitempty"`
}

// MilestonesResponse is the response from listing milestones
type MilestonesResponse struct {
	Milestones []Milestone `json:"milestones"`
}

// CreateMilestoneRequest is the request body for creating a milestone
type CreateMilestoneRequest struct {
	Title           string  `json:"title"`
	Description     string  `json:"description,omitempty"`
	App             string  `json:"app,omitempty"`
	Repo            string  `json:"repo,omitempty"`
	Branch          string  `json:"branch,omitempty"`
	Workflow        string  `json:"workflow,omitempty"`
	GithubMilestone int     `json:"github_milestone,omitempty"`
	Assignees       string  `json:"assignee_ids,omitempty"` // comma-separated IDs
	Project         string  `json:"project,omitempty"`
	AIModel         *string `json:"ai_model,omitempty"`
}

// UpdateMilestoneRequest is the request body for updating a milestone
type UpdateMilestoneRequest struct {
	Title           string  `json:"title,omitempty"`
	Description     string  `json:"description,omitempty"`
	App             string  `json:"app,omitempty"`
	Repo            string  `json:"repo,omitempty"`
	Branch          string  `json:"branch,omitempty"`
	Workflow        string  `json:"workflow,omitempty"`
	GithubMilestone int     `json:"github_milestone,omitempty"`
	Project         string  `json:"project,omitempty"`
	AIModel         *string `json:"ai_model,omitempty"`
}

// IncidentList is one incident type's listing. The server passes
// AppSignal's own text (Markdown) through as Content, like every other
// AppSignal endpoint.
type IncidentList struct {
	Type    string `json:"type"` // "exception" or "anomaly"
	Content string `json:"content"`
}

// IncidentLists holds the listings of an incidents response, exceptions
// before anomalies.
type IncidentLists []IncidentList

// ErrUnexpectedIncidents means the server's "incidents" value has neither
// shape below. It is an error, not an empty result, so an unknown shape
// never reads as "no incidents".
var ErrUnexpectedIncidents = errors.New("unexpected incidents response")

// UnmarshalJSON accepts both shapes the server sends under "incidents": one
// listing ({"type","content"}) for --type exception or anomaly, or one per
// type ({"exceptions":{...},"anomalies":{...}}) for all types.
func (l *IncidentLists) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return ErrUnexpectedIncidents
	}
	var v struct {
		IncidentList
		Exceptions *IncidentList `json:"exceptions"`
		Anomalies  *IncidentList `json:"anomalies"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	var lists IncidentLists
	switch {
	case v.Exceptions != nil || v.Anomalies != nil:
		for _, list := range []*IncidentList{v.Exceptions, v.Anomalies} {
			if list != nil {
				lists = append(lists, *list)
			}
		}
	case v.Type != "":
		lists = IncidentLists{v.IncidentList}
	default:
		return ErrUnexpectedIncidents
	}
	for _, list := range lists {
		if list.Type == "" {
			return ErrUnexpectedIncidents
		}
	}
	*l = lists
	return nil
}

// IncidentsResponse is the response from listing incidents
type IncidentsResponse struct {
	Incidents IncidentLists `json:"incidents"`
}

// ErrUnexpectedIncident means the server's incident response is not
// {"number","content"}.
var ErrUnexpectedIncident = errors.New("unexpected incident response")

// IncidentDetail is one incident. The server passes AppSignal's own text
// (Markdown) through as Content.
type IncidentDetail struct {
	Number  int    `json:"number"`
	Content string `json:"content"`
}

// UnmarshalJSON requires both fields, so an unknown shape is an error and
// never prints as an empty incident.
func (d *IncidentDetail) UnmarshalJSON(data []byte) error {
	var v struct {
		Number  *int    `json:"number"`
		Content *string `json:"content"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	if v.Number == nil || v.Content == nil {
		return ErrUnexpectedIncident
	}
	*d = IncidentDetail{Number: *v.Number, Content: *v.Content}
	return nil
}

// AssignIssueRequest is the request body for assigning an issue
type AssignIssueRequest struct {
	AssigneeIDs string `json:"assignee_ids"`
	Project     string `json:"project,omitempty"`
}

// UpdateIncidentRequest is the request body for updating incidents
type UpdateIncidentRequest struct {
	State    string `json:"state,omitempty"`
	Severity string `json:"severity,omitempty"`
	Assign   string `json:"assign,omitempty"`   // comma-separated user IDs
	Unassign string `json:"unassign,omitempty"` // comma-separated user IDs
}

// ErrUnexpectedResources means the server's resources response is not
// {"sections","content"}.
var ErrUnexpectedResources = errors.New("unexpected resources response")

// AppSignalResources is an app's resources (users, notifiers, namespaces,
// dashboards). The server passes AppSignal's own text (Markdown) through as
// Content; Sections lists the sections it asked for.
type AppSignalResources struct {
	Sections []string `json:"sections"`
	Content  string   `json:"content"`
}

// UnmarshalJSON requires both fields, as for IncidentDetail.
func (r *AppSignalResources) UnmarshalJSON(data []byte) error {
	var v struct {
		Sections []string `json:"sections"`
		Content  *string  `json:"content"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	if v.Sections == nil || v.Content == nil {
		return ErrUnexpectedResources
	}
	*r = AppSignalResources{Sections: v.Sections, Content: *v.Content}
	return nil
}

// ListIssuesOptions contains options for listing issues
type ListIssuesOptions struct {
	Stage     string
	Context   string
	App       string
	Completed bool
	Project   string
}

// ListEpicsOptions contains options for listing epics
type ListEpicsOptions struct {
	Milestone string
	Completed bool
	Project   string
}

// ListMilestonesOptions contains options for listing milestones
type ListMilestonesOptions struct {
	Completed bool
	App       string
	Project   string
}

// ListIncidentsOptions contains options for listing incidents
type ListIncidentsOptions struct {
	App       string
	Type      string // "exception", "anomaly", or "all"
	State     string
	Namespace string
	Start     string
	End       string
}

// Notification is one of the signed-in person's Basecamp notifications.
type Notification struct {
	ID           int64   `json:"id"`
	Type         string  `json:"type"`
	Title        string  `json:"title"`
	Excerpt      *string `json:"excerpt"`
	Creator      *string `json:"creator"`
	CreatedAt    string  `json:"created_at"`
	ReadAt       *string `json:"read_at"`
	URL          string  `json:"url"`
	FluxID       *string `json:"flux_id"`
	CommentID    *string `json:"comment_id"`
	ReadableSGID string  `json:"readable_sgid"`
}
