package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrUnauthorized = errors.New("unauthorized: run 'flux auth login' to sign in")
	ErrNotFound     = errors.New("not found")
	// ErrNotSupported means the server has no route for the request, so it
	// is older than this CLI.
	ErrNotSupported = errors.New("not supported by this server: it is older than this CLI; upgrade the Flux server")
	// ErrProductNotConfigured means the project has no Product to-do set, so
	// it has no scenes or use cases.
	ErrProductNotConfigured = errors.New("product_not_configured: this project has no Product (scenes and use cases) configured on the server; pick another project with --project")
	// ErrAmbiguousCode means a code matches several items; the error names
	// their IDs.
	ErrAmbiguousCode = errors.New("ambiguous_code")
)

const (
	deliveryPrefix       = "/api/delivery"
	legacyDeliveryPrefix = "/api/dev"
	productPrefix        = "/api/product"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	// legacyDelivery is set once the server turns out to predate
	// /api/delivery; delivery requests then go to /api/dev.
	legacyDelivery bool
}

type issueFields struct {
	ID          string         `json:"id"`
	Ref         metadataString `json:"ref,omitempty"`
	Title       string         `json:"title"`
	Stage       string         `json:"stage"`
	SubStage    string         `json:"sub_stage,omitempty"`
	Context     string         `json:"context,omitempty"`
	Program     string         `json:"program,omitempty"` // deprecated alias of Context, from older servers
	Size        string         `json:"size,omitempty"`
	Priority    int            `json:"priority,omitempty"`
	App         string         `json:"app,omitempty"`
	Epic        metadataString `json:"epic,omitempty"`
	Milestone   metadataString `json:"milestone,omitempty"`
	UseCase     metadataString `json:"use_case,omitempty"`
	URL         string         `json:"url,omitempty"`
	Completed   bool           `json:"completed"`
	Description string         `json:"description,omitempty"`
	Assignees   []Person       `json:"assignees,omitempty"`
	Thread      []Comment      `json:"thread,omitempty"`
}

// Issue is an issue as the server returns it. JSON output repeats the
// server's own JSON, so it keeps every field the server sends.
type Issue struct {
	issueFields
	raw json.RawMessage
}

// DisplayContext is the issue's context, the bracketed title prefix. Older
// servers only report it under its deprecated name, program.
func (i Issue) DisplayContext() string {
	if i.Context != "" {
		return i.Context
	}
	return i.Program
}

func (i *Issue) UnmarshalJSON(data []byte) error {
	return unmarshalRaw(data, &i.issueFields, &i.raw)
}

func (i Issue) MarshalJSON() ([]byte, error) {
	return marshalRaw(i.raw, i.issueFields)
}

// unmarshalRaw decodes data into fields and keeps a copy of data in raw.
func unmarshalRaw(data []byte, fields interface{}, raw *json.RawMessage) error {
	if err := json.Unmarshal(data, fields); err != nil {
		return err
	}
	*raw = append(json.RawMessage(nil), data...)
	return nil
}

// marshalRaw returns the kept server JSON, or fields when there is none.
func marshalRaw(raw json.RawMessage, fields interface{}) ([]byte, error) {
	if len(raw) > 0 && string(raw) != "null" {
		return raw, nil
	}
	return json.Marshal(fields)
}

// metadataString decodes Basecamp metadata values that the server may send
// as a string, a number or null.
type metadataString string

func (m *metadataString) UnmarshalJSON(data []byte) error {
	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	switch v := value.(type) {
	case nil:
		*m = ""
	case string:
		*m = metadataString(v)
	default:
		*m = metadataString(string(data))
	}
	return nil
}

type Comment struct {
	ID       string  `json:"id"`
	Author   string  `json:"author"`
	AuthorID *string `json:"author_id,omitempty"`
	Date     string  `json:"date"`
	Content  string  `json:"content"`
}

type AdvanceResult struct {
	TargetStage    string `json:"target_stage"`
	TargetSubstage string `json:"target_substage,omitempty"`
	StageCommentID string `json:"stage_comment_id"`
	UserCommentID  string `json:"user_comment_id,omitempty"`
}

type Persona struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Role         string `json:"role"`
	Signature    string `json:"signature"`
	SystemPrompt string `json:"system_prompt,omitempty"`
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

// doRequest performs an HTTP request with optional authentication
func (c *Client) doRequest(method, endpoint string, body interface{}) (*http.Response, error) {
	var jsonData []byte
	if body != nil {
		var err error
		jsonData, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	return c.send(endpoint, func(endpoint string) (*http.Request, error) {
		var reqBody io.Reader
		if body != nil {
			reqBody = bytes.NewReader(jsonData)
		}
		req, err := http.NewRequest(method, c.baseURL+endpoint, reqBody)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		return req, nil
	})
}

// send builds and performs a request for endpoint. Delivery endpoints go to
// /api/delivery; when the server has no such route, the request is retried
// once on /api/dev, and later delivery requests go there directly.
func (c *Client) send(endpoint string, build func(endpoint string) (*http.Request, error)) (*http.Response, error) {
	delivery := strings.HasPrefix(endpoint, deliveryPrefix+"/")
	if delivery && c.legacyDelivery {
		endpoint = legacyDeliveryPrefix + strings.TrimPrefix(endpoint, deliveryPrefix)
	}

	req, err := build(endpoint)
	if err != nil {
		return nil, err
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil || !delivery || c.legacyDelivery || !routeMissing(resp) {
		return resp, err
	}

	resp.Body.Close()
	c.legacyDelivery = true
	return c.send(endpoint, build)
}

// routeMissing reports whether resp is a 404 for a route the server does not
// have. Flux's own 404s carry an "error" field; Phoenix's no-route 404 does
// not. The response body stays readable.
func routeMissing(resp *http.Response) bool {
	if resp.StatusCode != http.StatusNotFound {
		return false
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(body))

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return true
	}
	_, hasError := payload["error"]
	return !hasError
}

// get performs a GET request
func (c *Client) get(endpoint string) (*http.Response, error) {
	return c.doRequest(http.MethodGet, endpoint, nil)
}

// post performs a POST request with JSON body
func (c *Client) post(endpoint string, body interface{}) (*http.Response, error) {
	return c.doRequest(http.MethodPost, endpoint, body)
}

// patch performs a PATCH request with JSON body
func (c *Client) patch(endpoint string, body interface{}) (*http.Response, error) {
	return c.doRequest(http.MethodPatch, endpoint, body)
}

// delete performs a DELETE request
func (c *Client) delete(endpoint string) (*http.Response, error) {
	return c.doRequest(http.MethodDelete, endpoint, nil)
}

// handleResponseError handles common HTTP error responses
func (c *Client) handleResponseError(resp *http.Response, action string) error {
	if routeMissing(resp) {
		return ErrNotSupported
	}
	body, _ := io.ReadAll(resp.Body)

	var payload struct {
		Error   string   `json:"error"`
		Message string   `json:"message"`
		IDs     []string `json:"ids"`
	}
	json.Unmarshal(body, &payload) //nolint:errcheck
	switch payload.Error {
	case "product_not_configured":
		return ErrProductNotConfigured
	case "ambiguous_code":
		return ambiguousCodeError(payload.Message, payload.IDs)
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusNotFound:
		if payload.Error != "" && payload.Error != "Not Found" {
			return fmt.Errorf("%w: %s", ErrNotFound, payload.Error)
		}
		return ErrNotFound
	default:
		return fmt.Errorf("failed to %s (%d): %s", action, resp.StatusCode, string(body))
	}
}

// ambiguousCodeError names the IDs that share a code, so the caller can pick
// one.
func ambiguousCodeError(message string, ids []string) error {
	if message == "" {
		message = "the code matches several items; use an ID"
	}
	if len(ids) == 0 {
		return fmt.Errorf("%w: %s", ErrAmbiguousCode, message)
	}
	return fmt.Errorf("%w: %s. Matching IDs: %s", ErrAmbiguousCode, message, strings.Join(ids, ", "))
}

func (c *Client) GetIdentity() (*Identity, error) {
	if c.apiKey == "" {
		return nil, ErrUnauthorized
	}

	resp, err := c.get("/api/delivery/identity")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch identity: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleResponseError(resp, "fetch identity")
	}

	var identity Identity
	if err := json.NewDecoder(resp.Body).Decode(&identity); err != nil {
		return nil, fmt.Errorf("failed to decode identity: %w", err)
	}
	if identity.BasecampAccountID == "" || identity.BasecampPersonID == "" || identity.DisplayName == "" {
		return nil, errors.New("incomplete identity response")
	}

	return &identity, nil
}

type IssuesResponse struct {
	Issues []Issue `json:"issues"`
}

func (c *Client) ListIssues(opts ListIssuesOptions) ([]Issue, error) {
	params := url.Values{}
	if opts.Stage != "" {
		params.Set("stage", opts.Stage)
	}
	if opts.Context != "" {
		params.Set("context", opts.Context)
		// Older servers only know the deprecated name.
		params.Set("program", opts.Context)
	}
	if opts.App != "" {
		params.Set("app", opts.App)
	}
	if opts.Completed {
		params.Set("completed", "true")
	}
	if opts.Project != "" {
		params.Set("project", opts.Project)
	}

	endpoint := "/api/delivery/issues"
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	resp, err := c.get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch issues: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleResponseError(resp, "list issues")
	}

	var response IssuesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Issues, nil
}

func (c *Client) GetIssue(id, project string) (*Issue, error) {
	endpoint := fmt.Sprintf("/api/delivery/issues/%s", url.PathEscape(id))
	if project != "" {
		endpoint += "?" + url.Values{"project": {project}}.Encode()
	}

	resp, err := c.get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch issue: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleResponseError(resp, "fetch issue")
	}

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &issue, nil
}

type PersonasResponse struct {
	Personas []Persona `json:"personas"`
}

type ListPersonasOptions struct {
	Type          string // "dev", "conversation", or "all"
	IncludePrompt bool   // Include system_prompt in response
}

func (c *Client) ListPersonas(opts *ListPersonasOptions) ([]Persona, error) {
	endpoint := "/api/delivery/personas"

	params := url.Values{}
	if opts != nil {
		if opts.Type != "" {
			params.Set("type", opts.Type)
		}
		if opts.IncludePrompt {
			params.Set("include_prompt", "true")
		}
	}

	if len(params) > 0 {
		endpoint = endpoint + "?" + params.Encode()
	}

	resp, err := c.get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch personas: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleResponseError(resp, "fetch personas")
	}

	var response PersonasResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Personas, nil
}

// =============================================================================
// Projects
// =============================================================================

func (c *Client) ListProjects() ([]Project, error) {
	resp, err := c.get("/api/delivery/projects")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch projects: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleResponseError(resp, "list projects")
	}

	var response ProjectsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Projects, nil
}

// =============================================================================
// Issues - Extended
// =============================================================================

func (c *Client) CreateIssue(req CreateIssueRequest) (*Issue, error) {
	resp, err := c.post("/api/delivery/issues", req)
	if err != nil {
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.handleResponseError(resp, "create issue")
	}

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &issue, nil
}

func (c *Client) UpdateIssue(id string, req UpdateIssueRequest) (*Issue, error) {
	endpoint := fmt.Sprintf("/api/delivery/issues/%s", url.PathEscape(id))
	resp, err := c.patch(endpoint, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update issue: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleResponseError(resp, "update issue")
	}

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &issue, nil
}

func (c *Client) DeleteIssue(id, project string) error {
	endpoint := fmt.Sprintf("/api/delivery/issues/%s", url.PathEscape(id))
	if project != "" {
		endpoint += "?project=" + url.QueryEscape(project)
	}

	resp, err := c.delete(endpoint)
	if err != nil {
		return fmt.Errorf("failed to delete issue: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return c.handleResponseError(resp, "delete issue")
	}

	return nil
}

func (c *Client) AddIssueComment(id string, req CommentRequest) (*CommentResponse, error) {
	endpoint := fmt.Sprintf("/api/delivery/issues/%s/comments", url.PathEscape(id))
	resp, err := c.post(endpoint, req)
	if err != nil {
		return nil, fmt.Errorf("failed to add comment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.handleResponseError(resp, "add comment")
	}

	var result CommentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (c *Client) AdvanceIssue(id string, req AdvanceIssueRequest) (*AdvanceResult, error) {
	endpoint := fmt.Sprintf("/api/delivery/issues/%s/advance", url.PathEscape(id))
	resp, err := c.post(endpoint, req)
	if err != nil {
		return nil, fmt.Errorf("failed to advance issue: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleResponseError(resp, "advance issue")
	}

	var result AdvanceResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// LinkIssue links an issue to an epic, milestone or use case, or unlinks it,
// and returns the server's response.
func (c *Client) LinkIssue(id string, req LinkRequest) (json.RawMessage, error) {
	endpoint := fmt.Sprintf("/api/delivery/issues/%s/link", url.PathEscape(id))
	resp, err := c.post(endpoint, req)
	if err != nil {
		return nil, fmt.Errorf("failed to link issue: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleResponseError(resp, "link issue")
	}

	return decodeRawObject(resp.Body)
}

func (c *Client) AssignIssue(id string, req AssignIssueRequest) error {
	endpoint := fmt.Sprintf("/api/delivery/issues/%s/assign", url.PathEscape(id))
	resp, err := c.post(endpoint, req)
	if err != nil {
		return fmt.Errorf("failed to assign issue: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.handleResponseError(resp, "assign issue")
	}

	return nil
}

// =============================================================================
// Epics
// =============================================================================

func (c *Client) ListEpics(opts ListEpicsOptions) ([]Epic, error) {
	params := url.Values{}
	if opts.Milestone != "" {
		params.Set("milestone", opts.Milestone)
	}
	if opts.Completed {
		params.Set("completed", "true")
	}
	if opts.Project != "" {
		params.Set("project", opts.Project)
	}

	endpoint := "/api/delivery/epics"
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	resp, err := c.get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch epics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleResponseError(resp, "list epics")
	}

	var response EpicsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Epics, nil
}

func (c *Client) GetEpic(id, project string) (*Epic, error) {
	endpoint := fmt.Sprintf("/api/delivery/epics/%s", url.PathEscape(id))
	if project != "" {
		endpoint += "?project=" + url.QueryEscape(project)
	}

	resp, err := c.get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch epic: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleResponseError(resp, "get epic")
	}

	var epic Epic
	if err := json.NewDecoder(resp.Body).Decode(&epic); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &epic, nil
}

func (c *Client) CreateEpic(req CreateEpicRequest) (*Epic, error) {
	resp, err := c.post("/api/delivery/epics", req)
	if err != nil {
		return nil, fmt.Errorf("failed to create epic: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.handleResponseError(resp, "create epic")
	}

	var epic Epic
	if err := json.NewDecoder(resp.Body).Decode(&epic); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &epic, nil
}

func (c *Client) UpdateEpic(id string, req UpdateEpicRequest) (*Epic, error) {
	endpoint := fmt.Sprintf("/api/delivery/epics/%s", url.PathEscape(id))
	resp, err := c.patch(endpoint, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update epic: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleResponseError(resp, "update epic")
	}

	var epic Epic
	if err := json.NewDecoder(resp.Body).Decode(&epic); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &epic, nil
}

func (c *Client) ListEpicIssues(id string, completed bool, project string) ([]Issue, error) {
	params := url.Values{}
	if completed {
		params.Set("include_completed", "true")
	}
	if project != "" {
		params.Set("project", project)
	}

	endpoint := fmt.Sprintf("/api/delivery/epics/%s/issues", url.PathEscape(id))
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	resp, err := c.get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch epic issues: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleResponseError(resp, "list epic issues")
	}

	var response IssuesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Issues, nil
}

func (c *Client) AddEpicComment(id string, req CommentRequest) (*CommentResponse, error) {
	endpoint := fmt.Sprintf("/api/delivery/epics/%s/comments", url.PathEscape(id))
	resp, err := c.post(endpoint, req)
	if err != nil {
		return nil, fmt.Errorf("failed to add comment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.handleResponseError(resp, "add comment")
	}

	var result CommentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// =============================================================================
// Milestones
// =============================================================================

func (c *Client) ListMilestones(opts ListMilestonesOptions) ([]Milestone, error) {
	params := url.Values{}
	if opts.Completed {
		params.Set("completed", "true")
	}
	if opts.App != "" {
		params.Set("app", opts.App)
	}
	if opts.Project != "" {
		params.Set("project", opts.Project)
	}

	endpoint := "/api/delivery/milestones"
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	resp, err := c.get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch milestones: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleResponseError(resp, "list milestones")
	}

	var response MilestonesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Milestones, nil
}

func (c *Client) GetMilestone(id, project string) (*Milestone, error) {
	endpoint := fmt.Sprintf("/api/delivery/milestones/%s", url.PathEscape(id))
	if project != "" {
		endpoint += "?project=" + url.QueryEscape(project)
	}

	resp, err := c.get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch milestone: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleResponseError(resp, "get milestone")
	}

	var milestone Milestone
	if err := json.NewDecoder(resp.Body).Decode(&milestone); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &milestone, nil
}

func (c *Client) CreateMilestone(req CreateMilestoneRequest) (*Milestone, error) {
	resp, err := c.post("/api/delivery/milestones", req)
	if err != nil {
		return nil, fmt.Errorf("failed to create milestone: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.handleResponseError(resp, "create milestone")
	}

	var milestone Milestone
	if err := json.NewDecoder(resp.Body).Decode(&milestone); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &milestone, nil
}

func (c *Client) UpdateMilestone(id string, req UpdateMilestoneRequest) (*Milestone, error) {
	endpoint := fmt.Sprintf("/api/delivery/milestones/%s", url.PathEscape(id))
	resp, err := c.patch(endpoint, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update milestone: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleResponseError(resp, "update milestone")
	}

	var milestone Milestone
	if err := json.NewDecoder(resp.Body).Decode(&milestone); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &milestone, nil
}

func (c *Client) ListMilestoneIssues(id string, completed bool, project string) ([]Issue, error) {
	params := url.Values{}
	if completed {
		params.Set("include_completed", "true")
	}
	if project != "" {
		params.Set("project", project)
	}

	endpoint := fmt.Sprintf("/api/delivery/milestones/%s/issues", url.PathEscape(id))
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	resp, err := c.get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch milestone issues: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleResponseError(resp, "list milestone issues")
	}

	var response IssuesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Issues, nil
}

func (c *Client) AddMilestoneComment(id string, req CommentRequest) (*CommentResponse, error) {
	endpoint := fmt.Sprintf("/api/delivery/milestones/%s/comments", url.PathEscape(id))
	resp, err := c.post(endpoint, req)
	if err != nil {
		return nil, fmt.Errorf("failed to add comment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.handleResponseError(resp, "add comment")
	}

	var result CommentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// =============================================================================
// AppSignal
// =============================================================================

func (c *Client) ListAppSignalApps() ([]string, error) {
	resp, err := c.get("/api/delivery/appsignal/apps")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch apps: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleResponseError(resp, "list apps")
	}

	var response struct {
		Apps []string `json:"apps"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Apps, nil
}

func (c *Client) ListIncidents(opts ListIncidentsOptions) ([]Incident, error) {
	params := url.Values{}
	params.Set("app", opts.App)
	if opts.Type != "" {
		params.Set("type", opts.Type)
	}
	if opts.State != "" {
		params.Set("state", opts.State)
	}
	if opts.Namespace != "" {
		params.Set("namespace", opts.Namespace)
	}
	if opts.Start != "" {
		params.Set("start", opts.Start)
	}
	if opts.End != "" {
		params.Set("end", opts.End)
	}

	endpoint := "/api/delivery/appsignal/incidents?" + params.Encode()

	resp, err := c.get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch incidents: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleResponseError(resp, "list incidents")
	}

	var response IncidentsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Incidents, nil
}

func (c *Client) GetIncident(app string, number int) (*IncidentDetail, error) {
	endpoint := fmt.Sprintf("/api/delivery/appsignal/incidents/%d?app=%s", number, url.QueryEscape(app))

	resp, err := c.get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch incident: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleResponseError(resp, "get incident")
	}

	var incident IncidentDetail
	if err := json.NewDecoder(resp.Body).Decode(&incident); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &incident, nil
}

func (c *Client) UpdateIncidents(app, incidents string, req UpdateIncidentRequest) error {
	endpoint := fmt.Sprintf("/api/delivery/appsignal/incidents?app=%s&incidents=%s",
		url.QueryEscape(app), url.QueryEscape(incidents))

	resp, err := c.patch(endpoint, req)
	if err != nil {
		return fmt.Errorf("failed to update incidents: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.handleResponseError(resp, "update incidents")
	}

	return nil
}

func (c *Client) AddIncidentNote(app string, number int, content string) error {
	endpoint := fmt.Sprintf("/api/delivery/appsignal/incidents/%d/notes?app=%s",
		number, url.QueryEscape(app))

	req := struct {
		Content string `json:"content"`
	}{Content: content}

	resp, err := c.post(endpoint, req)
	if err != nil {
		return fmt.Errorf("failed to add note: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return c.handleResponseError(resp, "add note")
	}

	return nil
}

func (c *Client) GetAppSignalResources(app, sections string) (*AppSignalResources, error) {
	params := url.Values{}
	params.Set("app", app)
	if sections != "" {
		params.Set("sections", sections)
	}

	endpoint := "/api/delivery/appsignal/resources?" + params.Encode()

	resp, err := c.get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch resources: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleResponseError(resp, "get resources")
	}

	var resources AppSignalResources
	if err := json.NewDecoder(resp.Body).Decode(&resources); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &resources, nil
}

// =============================================================================
// People
// =============================================================================

type Person struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

func (c *Client) ListPeople(project string) ([]Person, error) {
	endpoint := "/api/delivery/people"
	if project != "" {
		endpoint += "?project=" + url.QueryEscape(project)
	}

	resp, err := c.get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch people: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleResponseError(resp, "list people")
	}

	var response struct {
		People []Person `json:"people"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.People, nil
}

// =============================================================================
// Comments
// =============================================================================

type CommentResult struct {
	ID string `json:"id"`
}

func (c *Client) UpdateComment(id string, req CommentRequest) (*CommentResult, error) {
	resp, err := c.patch(fmt.Sprintf("/api/delivery/comments/%s", url.PathEscape(id)), req)
	if err != nil {
		return nil, fmt.Errorf("failed to update comment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleResponseError(resp, "update comment")
	}

	var result CommentResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (c *Client) DeleteComment(id, project string) (*CommentResult, error) {
	endpoint := fmt.Sprintf("/api/delivery/comments/%s", url.PathEscape(id))
	if project != "" {
		endpoint += "?project=" + url.QueryEscape(project)
	}

	resp, err := c.delete(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to delete comment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleResponseError(resp, "delete comment")
	}

	var result CommentResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// =============================================================================
// Images & Diagrams
// =============================================================================

type AttachmentResult struct {
	SGID string `json:"sgid"`
	HTML string `json:"html"`
}

func (c *Client) UploadImage(filePath, caption, project string) (*AttachmentResult, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	fw, err := mw.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, err
	}
	if _, err = io.Copy(fw, f); err != nil {
		return nil, err
	}
	if caption != "" {
		mw.WriteField("caption", caption)
	}
	if project != "" {
		mw.WriteField("project", project)
	}
	mw.Close()

	resp, err := c.send("/api/delivery/images", func(endpoint string) (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, c.baseURL+endpoint, bytes.NewReader(buf.Bytes()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", mw.FormDataContentType())
		return req, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleResponseError(resp, "upload image")
	}

	var result AttachmentResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (c *Client) RenderDiagram(mermaid, caption, project string) (*AttachmentResult, error) {
	body := map[string]string{"mermaid": mermaid}
	if caption != "" {
		body["caption"] = caption
	}
	if project != "" {
		body["project"] = project
	}

	resp, err := c.post("/api/delivery/diagrams", body)
	if err != nil {
		return nil, fmt.Errorf("failed to render diagram: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleResponseError(resp, "render diagram")
	}

	var result AttachmentResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// =============================================================================
// Resync
// =============================================================================

type ResyncResult struct {
	Updated int `json:"updated"`
	Checked int `json:"checked"`
}

func (c *Client) ResyncEpic(id, project string) (*ResyncResult, error) {
	endpoint := fmt.Sprintf("/api/delivery/epics/%s/resync", url.PathEscape(id))
	body := map[string]string{}
	if project != "" {
		body["project"] = project
	}

	resp, err := c.post(endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to resync epic: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleResponseError(resp, "resync epic")
	}

	var result ResyncResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (c *Client) ResyncMilestone(id, project string) (*ResyncResult, error) {
	endpoint := fmt.Sprintf("/api/delivery/milestones/%s/resync", url.PathEscape(id))
	body := map[string]string{}
	if project != "" {
		body["project"] = project
	}

	resp, err := c.post(endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to resync milestone: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleResponseError(resp, "resync milestone")
	}

	var result ResyncResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
