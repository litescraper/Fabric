package serper

// Candidate for etheia-agent-map — no existing equivalent could be inspected.
// See: https://github.com/Etheia-Consulting-LLC/etheia-agent-map

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danielmiessler/fabric/internal/plugins"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

type Client struct {
	*plugins.PluginBase
	ApiKey  *plugins.SetupQuestion
	BaseURL *plugins.SetupQuestion
}

func NewClient() *Client {
	c := &Client{PluginBase: &plugins.PluginBase{
		Name: "Serper", SetupDescription: "Serper web search (optional)",
		EnvNamePrefix: plugins.BuildEnvVariablePrefix("Serper"),
	}}
	c.ApiKey = c.AddSetupQuestion("API Key", false)
	c.BaseURL = c.AddSetupQuestion("Base URL", false)
	c.BaseURL.Value = "https://google.serper.dev"
	c.ConfigureCustom = func() error {
		return plugins.RequireSecureBaseURL(c.BaseURL.Value)
	}
	return c
}

func (c *Client) IsConfigured() bool {
	return c.ApiKey.Value != ""
}

// Search returns titles, source URLs, and snippets, not scraped page content.
func (c *Client) Search(query string) (string, error) {
	if !c.IsConfigured() {
		return "", fmt.Errorf("SERPER_API_KEY is required")
	}
	if strings.TrimSpace(query) == "" {
		return "", fmt.Errorf("search query is required")
	}
	payload, err := json.Marshal(map[string]any{"q": query, "num": 10})
	if err != nil {
		return "", err
	}
	baseURL := c.BaseURL.Value
	if baseURL == "" {
		baseURL = "https://google.serper.dev"
	}
	if err := plugins.RequireSecureBaseURL(baseURL); err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/search", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("X-API-KEY", c.ApiKey.Value)
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("web search failed: HTTP %d", resp.StatusCode)
	}
	var result struct {
		Organic []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"organic"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Organic) == 0 {
		return "No web search results found.", nil
	}
	var output strings.Builder
	for _, r := range result.Organic {
		fmt.Fprintf(&output, "## %s\n%s\n\n%s\n\n", r.Title, r.Link, r.Snippet)
	}
	return strings.TrimSpace(output.String()), nil
}
