package api

// Project represents a Flux project
type Project struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// ProjectsResponse is the response from listing projects
type ProjectsResponse struct {
	Projects []Project `json:"projects"`
}

// CreateIssueRequest is the request body for creating an issue
type CreateIssueRequest struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Stage       string `json:"stage,omitempty"`
	Program     string `json:"program,omitempty"`
	Size        string `json:"size,omitempty"`
	Priority    int    `json:"priority,omitempty"`
	Epic        string `json:"epic,omitempty"`
	Milestone   string `json:"milestone,omitempty"`
	Persona     string `json:"persona,omitempty"`
	Project     string `json:"project,omitempty"`
}

// UpdateIssueRequest is the request body for updating an issue
type UpdateIssueRequest struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Size        string `json:"size,omitempty"`
	Priority    int    `json:"priority,omitempty"`
	Epic        string `json:"epic,omitempty"`
	Milestone   string `json:"milestone,omitempty"`
	Persona     string `json:"persona,omitempty"`
	Project     string `json:"project,omitempty"`
}

// AdvanceIssueRequest is the request body for advancing an issue stage
type AdvanceIssueRequest struct {
	TargetStage    string `json:"target_stage,omitempty"`
	TargetSubstage string `json:"target_substage,omitempty"`
	Comment        string `json:"comment,omitempty"`
	Persona        string `json:"persona,omitempty"`
	Project        string `json:"project,omitempty"`
}

// LinkRequest is the request body for linking/unlinking items
type LinkRequest struct {
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	Action     string `json:"action,omitempty"` // "link" or "unlink"
	Project    string `json:"project,omitempty"`
}

// CommentRequest is the request body for adding a comment
type CommentRequest struct {
	Content string `json:"content"`
	Persona string `json:"persona,omitempty"`
	Project string `json:"project,omitempty"`
}

// Epic represents an epic in the backlog
type Epic struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Description  string   `json:"description,omitempty"`
	Milestone    string   `json:"milestone,omitempty"`
	MilestoneID  string   `json:"milestone_id,omitempty"`
	Branch       string   `json:"branch,omitempty"`
	Assignees    []string `json:"assignees,omitempty"`
	LinkedIssues []string `json:"linked_issues,omitempty"`
	Thread       []Comment `json:"thread,omitempty"`
}

// EpicsResponse is the response from listing epics
type EpicsResponse struct {
	Epics []Epic `json:"epics"`
}

// CreateEpicRequest is the request body for creating an epic
type CreateEpicRequest struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Milestone   string `json:"milestone,omitempty"`
	Assignees   string `json:"assignee_ids,omitempty"` // comma-separated IDs
	Project     string `json:"project,omitempty"`
}

// UpdateEpicRequest is the request body for updating an epic
type UpdateEpicRequest struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Milestone   string `json:"milestone,omitempty"`
	Branch      string `json:"branch,omitempty"`
	Project     string `json:"project,omitempty"`
}

// LinkEpicRequest is the request body for linking an epic to a milestone
type LinkEpicRequest struct {
	MilestoneID string `json:"milestone_id"`
	Action      string `json:"action,omitempty"` // "link" or "unlink"
	Project     string `json:"project,omitempty"`
}

// Milestone represents a milestone in release planning
type Milestone struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Description     string   `json:"description,omitempty"`
	Repo            string   `json:"repo,omitempty"`
	Branch          string   `json:"branch,omitempty"`
	Workflow        string   `json:"workflow,omitempty"`
	GithubMilestone int      `json:"github_milestone,omitempty"`
	Assignees       []string `json:"assignees,omitempty"`
	LinkedEpics     []string `json:"linked_epics,omitempty"`
	LinkedIssues    []string `json:"linked_issues,omitempty"`
	Thread          []Comment `json:"thread,omitempty"`
}

// MilestonesResponse is the response from listing milestones
type MilestonesResponse struct {
	Milestones []Milestone `json:"milestones"`
}

// CreateMilestoneRequest is the request body for creating a milestone
type CreateMilestoneRequest struct {
	Title           string `json:"title"`
	Description     string `json:"description,omitempty"`
	Repo            string `json:"repo,omitempty"`
	Branch          string `json:"branch,omitempty"`
	Workflow        string `json:"workflow,omitempty"`
	GithubMilestone int    `json:"github_milestone,omitempty"`
	Assignees       string `json:"assignee_ids,omitempty"` // comma-separated IDs
	Project         string `json:"project,omitempty"`
}

// UpdateMilestoneRequest is the request body for updating a milestone
type UpdateMilestoneRequest struct {
	Title           string `json:"title,omitempty"`
	Description     string `json:"description,omitempty"`
	Repo            string `json:"repo,omitempty"`
	Branch          string `json:"branch,omitempty"`
	Workflow        string `json:"workflow,omitempty"`
	GithubMilestone int    `json:"github_milestone,omitempty"`
	Project         string `json:"project,omitempty"`
}

// Incident represents an AppSignal incident
type Incident struct {
	Number   int    `json:"number"`
	Name     string `json:"name"`
	State    string `json:"state"`
	Severity string `json:"severity,omitempty"`
	Type     string `json:"type,omitempty"`
	Count    int    `json:"count,omitempty"`
}

// IncidentsResponse is the response from listing incidents
type IncidentsResponse struct {
	Incidents []Incident `json:"incidents"`
}

// IncidentDetail represents detailed information about an incident
type IncidentDetail struct {
	Number      int      `json:"number"`
	Name        string   `json:"name"`
	State       string   `json:"state"`
	Severity    string   `json:"severity,omitempty"`
	Type        string   `json:"type,omitempty"`
	Message     string   `json:"message,omitempty"`
	StackTrace  string   `json:"stack_trace,omitempty"`
	Namespace   string   `json:"namespace,omitempty"`
	Assignees   []string `json:"assignees,omitempty"`
	Occurrences int      `json:"occurrences,omitempty"`
	Notes       []Note   `json:"notes,omitempty"`
}

// Note represents a note on an incident
type Note struct {
	Author  string `json:"author"`
	Date    string `json:"date"`
	Content string `json:"content"`
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

// AppSignalResources represents available resources for an AppSignal app
type AppSignalResources struct {
	Users      []AppSignalUser      `json:"users,omitempty"`
	Notifiers  []AppSignalNotifier  `json:"notifiers,omitempty"`
	Namespaces []string             `json:"namespaces,omitempty"`
	Dashboards []AppSignalDashboard `json:"dashboards,omitempty"`
}

// AppSignalUser represents a user in AppSignal
type AppSignalUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// AppSignalNotifier represents a notifier in AppSignal
type AppSignalNotifier struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// AppSignalDashboard represents a dashboard in AppSignal
type AppSignalDashboard struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListIssuesOptions contains options for listing issues
type ListIssuesOptions struct {
	Stage     string
	Program   string
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
