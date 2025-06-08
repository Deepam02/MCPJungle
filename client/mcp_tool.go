package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ToolInputSchema defines the schema for the input parameters of a tool
type ToolInputSchema struct {
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties,omitempty"`
	Required   []string       `json:"required,omitempty"`
}

// Tool represents a tool provided by an MCP Server registered in the registry.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema ToolInputSchema `json:"input_schema"`
}

type ToolInvokeResult struct {
	Meta    map[string]any   `json:"_meta,omitempty"`
	IsError bool             `json:"isError,omitempty"`
	Content []map[string]any `json:"content"`
}

// ListTools fetches the list of tools, optionally filtered by server name.
func (c *Client) ListTools(server string) ([]*Tool, error) {
	u, _ := c.constructAPIEndpoint("/tools")
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	if server != "" {
		q := req.URL.Query()
		q.Add("server", server)
		req.URL.RawQuery = q.Encode()
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to %s: %w", req.URL.String(), err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status: %d, message: %s", resp.StatusCode, body)
	}

	var tools []*Tool
	if err := json.NewDecoder(resp.Body).Decode(&tools); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return tools, nil
}

// GetTool fetches a specific tool by its name.
func (c *Client) GetTool(name string) (*Tool, error) {
	u, _ := c.constructAPIEndpoint("/tool")
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	q := req.URL.Query()
	q.Add("name", name)
	req.URL.RawQuery = q.Encode()

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to %s: %w", req.URL.String(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status: %d, message: %s", resp.StatusCode, body)
	}

	var tool Tool
	if err := json.NewDecoder(resp.Body).Decode(&tool); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &tool, nil
}

// InvokeTool sends a JSON payload to invoke a tool.
// For now, this function only supports invoking tools that return a string response.
func (c *Client) InvokeTool(name string, input map[string]any) (*ToolInvokeResult, error) {
	// We need to insert the tool name into the POST payload
	// In order not to mutate the user-supplied input, create a shallow copy of the input
	// and add the name field to it.
	payload := make(map[string]any, len(input)+1)
	for k, v := range input {
		payload[k] = v
	}
	payload["name"] = name

	body, _ := json.Marshal(payload)
	u, _ := c.constructAPIEndpoint("/tools/invoke")
	resp, err := c.HTTPClient.Post(u, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("request to server failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status: %d, message: %s", resp.StatusCode, string(respBody))
	}

	var result *ToolInvokeResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}
