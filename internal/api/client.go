package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type Issue struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Stage       string    `json:"stage"`
	SubStage    string    `json:"sub_stage,omitempty"`
	Program     string    `json:"program,omitempty"`
	Size        string    `json:"size,omitempty"`
	Priority    int       `json:"priority,omitempty"`
	Description string    `json:"description,omitempty"`
	Thread      []Comment `json:"thread,omitempty"`
}

type Comment struct {
	Author  string `json:"author"`
	Date    string `json:"date"`
	Content string `json:"content"`
}

type Persona struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Role      string `json:"role"`
	Signature string `json:"signature"`
}

func NewClient(env string) *Client {
	baseURL := "https://eyra-flux.fly.dev"
	if env == "test" {
		baseURL = "https://eyra-flux-test.fly.dev"
	}

	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

type IssuesResponse struct {
	Issues []Issue `json:"issues"`
}

func (c *Client) ListIssues(stage string) ([]Issue, error) {
	endpoint := "/api/dev/issues"
	if stage != "" {
		endpoint = fmt.Sprintf("%s?stage=%s", endpoint, url.QueryEscape(stage))
	}

	resp, err := c.httpClient.Get(c.baseURL + endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch issues: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	var response IssuesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Issues, nil
}

func (c *Client) GetIssue(id string) (*Issue, error) {
	endpoint := fmt.Sprintf("/api/dev/issues/%s", url.PathEscape(id))

	resp, err := c.httpClient.Get(c.baseURL + endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch issue: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
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

func (c *Client) ListPersonas() ([]Persona, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/dev/personas")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch personas: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	var response PersonasResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Personas, nil
}
