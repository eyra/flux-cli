package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// ProductKind names a Product entity: scenes or use cases. Its value is the
// path segment under /api/product.
type ProductKind string

const (
	KindScene   ProductKind = "scenes"
	KindUseCase ProductKind = "use_cases"
)

// Response is a decoded server response that marshals back to the server's
// exact JSON, so --json output keeps every field.
type Response[T any] struct {
	Value T
	raw   json.RawMessage
}

func (r *Response[T]) UnmarshalJSON(data []byte) error {
	return unmarshalRaw(data, &r.Value, &r.raw)
}

func (r Response[T]) MarshalJSON() ([]byte, error) {
	return marshalRaw(r.raw, r.Value)
}

type productItemFields struct {
	ID            string           `json:"id"`
	Code          string           `json:"code,omitempty"`
	Title         string           `json:"title"`
	Name          string           `json:"name,omitempty"`
	Status        string           `json:"status,omitempty"`
	Completed     bool             `json:"completed"`
	CommentsCount int              `json:"comments_count"`
	URL           string           `json:"url,omitempty"`
	Scene         metadataString   `json:"scene,omitempty"`
	UseCasesCount *int             `json:"use_cases_count,omitempty"`
	IssuesCount   *int             `json:"issues_count,omitempty"`
	Project       string           `json:"project,omitempty"`
	Description   string           `json:"description,omitempty"`
	Thread        []ProductComment `json:"thread,omitempty"`
	LinkedUseCase []LinkedIssue    `json:"linked_use_cases,omitempty"`
	LinkedIssues  []LinkedIssue    `json:"linked_issues,omitempty"`
}

// ProductItem is a scene or use case, as a list item or with its details.
type ProductItem struct {
	productItemFields
	raw json.RawMessage
}

func (p *ProductItem) UnmarshalJSON(data []byte) error {
	return unmarshalRaw(data, &p.productItemFields, &p.raw)
}

func (p ProductItem) MarshalJSON() ([]byte, error) {
	return marshalRaw(p.raw, p.productItemFields)
}

// ProductComment is a comment in a scene's or use case's thread.
type ProductComment struct {
	ID          string  `json:"id"`
	Author      string  `json:"author"`
	AuthorID    *string `json:"author_id,omitempty"`
	Date        string  `json:"date"`
	ContentHTML string  `json:"content_html"`
	URL         string  `json:"url,omitempty"`
}

// ProductRef is the short form of a scene or use case.
type ProductRef struct {
	ID    string `json:"id"`
	Code  string `json:"code,omitempty"`
	Title string `json:"title"`
	URL   string `json:"url,omitempty"`
}

// ProductWrite is the response to creating or updating a scene or use case.
type ProductWrite struct {
	ID    string         `json:"id"`
	Code  string         `json:"code,omitempty"`
	Title string         `json:"title"`
	URL   string         `json:"url,omitempty"`
	Scene metadataString `json:"scene,omitempty"`
}

// ProductCommentResult is the response to commenting on a scene or use case.
type ProductCommentResult struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	URL     string `json:"url,omitempty"`
}

// SceneUseCases lists the use cases linked to a scene.
type SceneUseCases struct {
	Scene    ProductRef    `json:"scene"`
	UseCases []ProductItem `json:"use_cases"`
}

// UseCaseIssues lists the issues linked to a use case.
type UseCaseIssues struct {
	UseCase ProductRef `json:"use_case"`
	Issues  []Issue    `json:"issues"`
}

// ProductCode is a suggested code for a new scene or use case.
type ProductCode struct {
	Code string `json:"code"`
}

// ResyncSummary is the response to refreshing a parent's linked titles.
type ResyncSummary struct {
	ParentType  string           `json:"parent_type"`
	ParentID    string           `json:"parent_id"`
	ParentTitle string           `json:"parent_title"`
	Checked     int              `json:"checked"`
	Updated     int              `json:"updated"`
	Updates     []ResyncUpdate   `json:"updates"`
	NotFound    []metadataString `json:"not_found"`
}

type ResyncUpdate struct {
	ID       string `json:"id"`
	OldTitle string `json:"old_title"`
	NewTitle string `json:"new_title"`
}

// CreateProductItemRequest is the request body for creating a scene or use case
type CreateProductItemRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	Code        string  `json:"code,omitempty"`
	Area        string  `json:"area,omitempty"`
	Status      string  `json:"status,omitempty"`
	Scene       string  `json:"scene,omitempty"` // use cases only
	AIModel     *string `json:"ai_model,omitempty"`
}

// UpdateProductItemRequest is the request body for updating a scene or use case
type UpdateProductItemRequest struct {
	Title       string  `json:"title,omitempty"`
	Description string  `json:"description,omitempty"`
	Code        string  `json:"code,omitempty"`
	Status      string  `json:"status,omitempty"`
	AIModel     *string `json:"ai_model,omitempty"`
}

// ProductCommentRequest is the request body for commenting on a scene or use case
type ProductCommentRequest struct {
	Content string  `json:"content"`
	AIModel *string `json:"ai_model,omitempty"`
}

// ProductLinkRequest is the request body for linking a use case to a scene
type ProductLinkRequest struct {
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	Action     string `json:"action,omitempty"` // "link" or "unlink"
}

// ListProductItemsOptions contains options for listing scenes or use cases
type ListProductItemsOptions struct {
	Completed bool
	Scene     string // use cases only: scene ID or code
	Project   string
}

// productEndpoint builds /api/product/<kind>[/<id>][/<suffix>]?<query>.
// The project goes in the query for every request, reads and writes alike.
func productEndpoint(kind ProductKind, id, suffix, project string, query url.Values) string {
	endpoint := productPrefix + "/" + string(kind)
	if id != "" {
		endpoint += "/" + url.PathEscape(id)
	}
	if suffix != "" {
		endpoint += "/" + suffix
	}
	if query == nil {
		query = url.Values{}
	}
	if project != "" {
		query.Set("project", project)
	}
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	return endpoint
}

// productCall performs a product request and decodes a success response
// (200 or 201) into out.
func (c *Client) productCall(method, endpoint string, body interface{}, action string, out interface{}) error {
	resp, err := c.doRequest(method, endpoint, body)
	if err != nil {
		return fmt.Errorf("failed to %s: %w", action, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return c.handleResponseError(resp, action)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	return nil
}

func (c *Client) ListProductItems(kind ProductKind, opts ListProductItemsOptions) ([]ProductItem, error) {
	query := url.Values{}
	if opts.Completed {
		query.Set("completed", "true")
	}
	if opts.Scene != "" {
		query.Set("scene", opts.Scene)
	}
	var response map[string][]ProductItem
	err := c.productCall(http.MethodGet, productEndpoint(kind, "", "", opts.Project, query), nil, "list "+kind.label()+"s", &response)
	if err != nil {
		return nil, err
	}
	return response[string(kind)], nil
}

func (c *Client) GetProductItem(kind ProductKind, id, project string, includeThread bool) (*ProductItem, error) {
	query := url.Values{}
	if !includeThread {
		query.Set("include_thread", "false")
	}
	var item ProductItem
	if err := c.productCall(http.MethodGet, productEndpoint(kind, id, "", project, query), nil, "get "+kind.label(), &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (c *Client) CreateProductItem(kind ProductKind, project string, req CreateProductItemRequest) (*Response[ProductWrite], error) {
	var result Response[ProductWrite]
	if err := c.productCall(http.MethodPost, productEndpoint(kind, "", "", project, nil), req, "create "+kind.label(), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UpdateProductItem(kind ProductKind, id, project string, req UpdateProductItemRequest) (*Response[ProductWrite], error) {
	var result Response[ProductWrite]
	if err := c.productCall(http.MethodPatch, productEndpoint(kind, id, "", project, nil), req, "update "+kind.label(), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) AddProductComment(kind ProductKind, id, project string, req ProductCommentRequest) (*Response[ProductCommentResult], error) {
	var result Response[ProductCommentResult]
	if err := c.productCall(http.MethodPost, productEndpoint(kind, id, "comments", project, nil), req, "add comment", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) ResyncProductItem(kind ProductKind, id, project string) (*Response[ResyncSummary], error) {
	var result Response[ResyncSummary]
	if err := c.productCall(http.MethodPost, productEndpoint(kind, id, "resync", project, nil), nil, "resync "+kind.label(), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) ListSceneUseCases(id, project string) (*Response[SceneUseCases], error) {
	var result Response[SceneUseCases]
	if err := c.productCall(http.MethodGet, productEndpoint(KindScene, id, "use_cases", project, nil), nil, "list scene use cases", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) ListUseCaseIssues(id, project string) (*Response[UseCaseIssues], error) {
	var result Response[UseCaseIssues]
	if err := c.productCall(http.MethodGet, productEndpoint(KindUseCase, id, "issues", project, nil), nil, "list use case issues", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// NextProductCode returns the next free code in area.
func (c *Client) NextProductCode(kind ProductKind, area, project string) (*Response[ProductCode], error) {
	var result Response[ProductCode]
	query := url.Values{"area": {area}}
	if err := c.productCall(http.MethodGet, productEndpoint(kind, "next_code", "", project, query), nil, "get next "+kind.label()+" code", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// LinkUseCase links a use case to a scene, or unlinks it.
func (c *Client) LinkUseCase(id, project string, req ProductLinkRequest) (json.RawMessage, error) {
	var result json.RawMessage
	if err := c.productCall(http.MethodPost, productEndpoint(KindUseCase, id, "link", project, nil), req, "link use case", &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (k ProductKind) label() string {
	if k == KindUseCase {
		return "use case"
	}
	return "scene"
}

// decodeRawObject reads a JSON object response and returns it unchanged.
func decodeRawObject(r io.Reader) (json.RawMessage, error) {
	var result json.RawMessage
	if err := json.NewDecoder(r).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return result, nil
}
