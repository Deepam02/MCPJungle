package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/duaraghav8/mcpjungle/internal/model"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// ListTools returns all tools registered in the registry.
func (m *MCPService) ListTools() ([]model.Tool, error) {
	var tools []model.Tool
	if err := m.db.Find(&tools).Error; err != nil {
		return nil, err
	}
	// prepend server name to tool names to ensure we only return the unique names of tools to user
	for i := range tools {
		var s model.McpServer
		if err := m.db.First(&s, "id = ?", tools[i].ServerID).Error; err != nil {
			return nil, fmt.Errorf("failed to get server for tool %s: %w", tools[i].Name, err)
		}
		tools[i].Name = mergeServerToolNames(s.Name, tools[i].Name)
	}
	return tools, nil
}

// ListToolsByServer fetches tools provided by an MCP server from the registry.
func (m *MCPService) ListToolsByServer(name string) ([]model.Tool, error) {
	if err := validateServerName(name); err != nil {
		return nil, err
	}

	s, err := m.GetMcpServer(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP server %s from DB: %w", name, err)
	}

	var tools []model.Tool
	if err := m.db.Where("server_id = ?", s.ID).Find(&tools).Error; err != nil {
		return nil, fmt.Errorf("failed to get tools for server %s from DB: %w", name, err)
	}

	// prepend server name to tool names to ensure we only return the unique names of tools to user
	for i := range tools {
		tools[i].Name = mergeServerToolNames(s.Name, tools[i].Name)
	}

	return tools, nil
}

func (m *MCPService) GetTool(name string) (*model.Tool, error) {
	serverName, toolName, ok := splitServerToolName(name)
	if !ok {
		return nil, fmt.Errorf("invalid input: tool name does not contain a %s separator", serverToolNameSep)
	}

	s, err := m.GetMcpServer(serverName)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP server %s from DB: %w", serverName, err)
	}

	var tool model.Tool
	if err := m.db.Where("server_id = ? AND name = ?", s.ID, toolName).First(&tool).Error; err != nil {
		return nil, fmt.Errorf("failed to get tool %s from DB: %w", name, err)
	}
	// set the tool name back to the full name including server name
	tool.Name = name
	return &tool, nil
}

// InvokeTool invokes a tool from a registered MCP server and returns its response.
func (m *MCPService) InvokeTool(ctx context.Context, name string, args map[string]any) (string, error) {
	serverName, toolName, ok := splitServerToolName(name)
	if !ok {
		return "", fmt.Errorf("invalid input: tool name does not contain a %s separator", serverToolNameSep)
	}

	serverModel, err := m.GetMcpServer(serverName)
	if err != nil {
		return "", fmt.Errorf(
			"failed to get details about MCP server %s from DB: %w",
			serverName,
			err,
		)
	}

	mcpClient, err := createMcpServerConn(ctx, serverModel.URL)
	if err != nil {
		return "", fmt.Errorf(
			"failed to create connection to MCP server %s: %w", serverName, err,
		)
	}
	defer mcpClient.Close()

	callToolReq := mcp.CallToolRequest{}
	callToolReq.Params.Name = toolName
	callToolReq.Params.Arguments = args

	// CallTool() doesn't return an error if the tool is not found.
	// Instead, it returns a "not found" message in the response.
	// TODO: detect this and return an error if the tool is not found.
	callToolResp, err := mcpClient.CallTool(ctx, callToolReq)
	if err != nil {
		return "", fmt.Errorf(
			"failed to call tool %s on MCP server %s: %w",
			toolName,
			serverName,
			err,
		)
	}

	textContent, ok := callToolResp.Content[0].(mcp.TextContent)
	if !ok {
		return "", errors.New("failed to get text content from tool response")
	}
	return textContent.Text, nil
}

// registerServerTools fetches all tools from an MCP server and registers them in the DB.
func (m *MCPService) registerServerTools(ctx context.Context, s *model.McpServer, c *client.Client) error {
	// fetch all tools from the server so they can be added to the DB
	resp, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return fmt.Errorf("failed to fetch tools from MCP server %s: %w", s.Name, err)
	}
	for _, tool := range resp.Tools {
		// extracting json schema is currently on best-effort basis
		// if it fails, we log the error and continue with the next tool
		jsonSchema, _ := json.Marshal(tool.InputSchema)

		t := &model.Tool{
			ServerID:    s.ID,
			Name:        tool.GetName(),
			Description: tool.Description,
			InputSchema: jsonSchema,
		}
		if err := m.db.Create(t).Error; err != nil {
			// TODO: Add error log about this failure
			// If registration of a tool fails, we should not fail the entire server registration.
			// Instead, continue with the next tool.

			//fmt.Printf("failed to register tool %s in DB: %w", mergeServerToolNames(s.Name, t.Name), err)
		} else {
			// Set tool name to include the server name prefix to make it recognizable by MCPJungle
			tool.Name = mergeServerToolNames(s.Name, tool.Name)
			// add the tool to the MCP proxy server
			m.mcpProxyServer.AddTool(tool, m.mcpProxyToolCallHandler)
		}
	}
	return nil
}

// deregisterServerTools deletes all tools that belong to an MCP server from the DB.
// It also removes the tools from the MCP proxy server.
func (m *MCPService) deregisterServerTools(s *model.McpServer) error {
	// load all tools for the server from the DB so we can delete them from the MCP proxy
	tools, err := m.ListToolsByServer(s.Name)
	if err != nil {
		return fmt.Errorf("failed to list tools for server %s: %w", s.Name, err)
	}

	// now it's safe to delete the server's tools from the DB
	if err := m.db.Where("server_id = ?", s.ID).Delete(&model.Tool{}).Error; err != nil {
		return fmt.Errorf("failed to delete tools for server %s: %w", s.Name, err)
	}

	// delete tools from MCP proxy server
	toolNames := make([]string, len(tools), len(tools))
	for i, tool := range tools {
		toolNames[i] = tool.Name
	}
	m.mcpProxyServer.DeleteTools(toolNames...)

	return nil
}
