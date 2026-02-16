package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/wricardo/mcp-training/statefullgame/game/engine"
	"github.com/wricardo/mcp-training/statefullgame/game/service"
)

func TestNewClient(t *testing.T) {
	baseURL := "http://localhost:8080"
	client := NewClient(baseURL)

	if client == nil {
		t.Fatal("Expected client to be created")
	}

	if client.baseURL != baseURL {
		t.Errorf("Expected baseURL %s, got %s", baseURL, client.baseURL)
	}

	if client.httpClient == nil {
		t.Error("Expected HTTP client to be initialized")
	}

	if client.mcpServer == nil {
		t.Error("Expected MCP server to be initialized")
	}
}

func TestClient_Run(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock response for API calls
		resp := map[string]interface{}{
			"id":        "test-session",
			"battery":   50,
			"score":     0,
			"game_over": false,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	if client == nil {
		t.Fatal("Failed to create client")
	}

	// Test that Run doesn't panic (we can't easily test the actual MCP behavior without complex setup)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Run() panicked: %v", r)
		}
	}()

	// We can't test Run() fully as it blocks, but we can test that the MCP server is properly initialized
	if client.mcpServer == nil {
		t.Error("MCP server should be initialized")
	}
}

func TestClient_apiCall(t *testing.T) {
	// Create a test server that returns a known response
	expectedResponse := map[string]interface{}{
		"id":        "test-session",
		"battery":   75,
		"score":     5,
		"game_over": false,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()

	client := NewClient(server.URL)

	var response map[string]interface{}
	err := client.apiCall("GET", "/api", nil, &response)
	if err != nil {
		t.Fatalf("apiCall failed: %v", err)
	}

	// Check that we got the expected response
	if response["id"] != expectedResponse["id"] {
		t.Errorf("Expected id %v, got %v", expectedResponse["id"], response["id"])
	}
}

func TestClient_apiCall_Error(t *testing.T) {
	client := NewClient("http://invalid-url-that-does-not-exist:9999")

	err := client.apiCall("GET", "/api", nil, nil)
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

func TestClient_apiCall_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := NewClient(server.URL)

	err := client.apiCall("GET", "/api", nil, nil)
	if err == nil {
		t.Error("Expected error for HTTP 500 response")
	}

	if !strings.Contains(err.Error(), "API error") {
		t.Errorf("Expected 'API error' in error message, got: %v", err)
	}
}

func TestClient_createSession(t *testing.T) {
	// Mock server that responds to session creation
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/sessions" {
			t.Errorf("Expected POST /api/sessions, got %s %s", r.Method, r.URL.Path)
		}

		resp := service.SessionInfo{
			ID:         "test-session-123",
			ConfigName: "classic",
			GameState: &engine.GameState{
				Battery: 50,
				Score:   0,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	// Test create session without config
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "create_session",
			Arguments: map[string]interface{}{},
		},
	}

	result, err := client.handleCreateSession(ctx, request)
	if err != nil {
		t.Fatalf("createSession failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Check that the result contains the session ID
	resultStr, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatal("Expected text content in result")
	}

	if !strings.Contains(resultStr.Text, "test-session-123") {
		t.Errorf("Expected session ID in result, got: %s", resultStr.Text)
	}
}

func TestFormatGameState(t *testing.T) {
	gameState := &engine.GameState{
		PlayerPos:  engine.Position{X: 5, Y: 3},
		Battery:    75,
		MaxBattery: 100,
		Score:      10,
		GameOver:   false,
		Victory:    false,
		Message:    "Welcome to the game!",
	}

	result := formatGameState(gameState)

	// Check that all important fields are included
	expectedFields := []string{
		"Position: (5,3)",
		"Battery: 75/100",
		"Score: 10",
		"Welcome to the game!",
	}

	for _, field := range expectedFields {
		if !strings.Contains(result, field) {
			t.Errorf("Expected field '%s' in formatted output, got: %s", field, result)
		}
	}
}

func TestFormatGameState_GameOver(t *testing.T) {
	gameState := &engine.GameState{
		PlayerPos:  engine.Position{X: 2, Y: 1},
		Battery:    0,
		MaxBattery: 50,
		Score:      5,
		GameOver:   true,
		Victory:    false,
		Message:    "Game over!",
	}

	result := formatGameState(gameState)

	if !strings.Contains(result, "💀 GAME OVER") {
		t.Errorf("Expected '💀 GAME OVER' in result, got: %s", result)
	}
}

func TestFormatGameState_Victory(t *testing.T) {
	gameState := &engine.GameState{
		PlayerPos:  engine.Position{X: 10, Y: 10},
		Battery:    25,
		MaxBattery: 50,
		Score:      15,
		GameOver:   true,
		Victory:    true,
		Message:    "Congratulations!",
	}

	result := formatGameState(gameState)

	if !strings.Contains(result, "🎉 VICTORY!") {
		t.Errorf("Expected '🎉 VICTORY!' in result, got: %s", result)
	}
}

func TestFormatMoveResult(t *testing.T) {
	moveResult := &service.MoveResult{
		Success: true,
		Message: "Moved successfully",
		GameState: &engine.GameState{
			PlayerPos: engine.Position{X: 3, Y: 4},
			Battery:   80,
			Score:     7,
		},
	}

	result := formatMoveResult(moveResult)

	expectedFields := []string{
		"✓ Move successful",
		"Position: (3,4)",
		"Battery: 80",
		"Score: 7",
	}

	for _, field := range expectedFields {
		if !strings.Contains(result, field) {
			t.Errorf("Expected field '%s' in formatted output, got: %s", field, result)
		}
	}
}

func TestFormatMoveResult_Failed(t *testing.T) {
	moveResult := &service.MoveResult{
		Success: false,
		Message: "Cannot move into wall",
		GameState: &engine.GameState{
			PlayerPos: engine.Position{X: 1, Y: 1},
			Battery:   60,
			Score:     3,
		},
	}

	result := formatMoveResult(moveResult)

	if !strings.Contains(result, "✗ Move failed") {
		t.Errorf("Expected '✗ Move failed' in result, got: %s", result)
	}

}

func TestClient_handleGameInstructions(t *testing.T) {
	client := NewClient("http://localhost:8080")
	ctx := context.Background()

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "game_instructions",
			Arguments: map[string]interface{}{},
		},
	}

	result, err := client.handleGameInstructions(ctx, request)
	if err != nil {
		t.Fatalf("handleGameInstructions failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Check that the result contains game instructions
	resultStr, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatal("Expected text content in result")
	}

	expectedContent := []string{
		"Tesla Road Trip Game - Complete Instructions",
		"GAME OBJECTIVE:",
		"GRID LEGEND:",
		"AI AGENTS - CRITICAL SUCCESS STRATEGIES:",
		"CHARACTER RECOGNITION (MOST COMMON FAILURE POINT)",
		"Parse Character-by-Character",
		"SYSTEMATIC WORLD MAPPING:",
		"CORRIDOR NAVIGATION TECHNIQUE:",
		"PROACTIVE BATTERY MANAGEMENT:",
		"SECTION-BASED PROBLEM SOLVING:",
		"CRITICAL PITFALLS TO AVOID:",
		"DEBUGGING CHARACTER RECOGNITION:",
		"MOVEMENT COMMANDS:",
		"CHARGING LOCATIONS:",
		"VICTORY CONDITIONS:",
		"Good luck navigating your Tesla Road Trip!",
	}

	for _, content := range expectedContent {
		if !strings.Contains(resultStr.Text, content) {
			t.Errorf("Expected '%s' in instructions, got: %s", content, resultStr.Text)
		}
	}
}

func TestClient_Integration(t *testing.T) {
	// Integration test that verifies the client can be created and initialized without errors
	client := NewClient("http://localhost:8080")

	if client == nil {
		t.Fatal("Failed to create client")
	}

	// Test that the MCP server has been properly configured with tools
	if client.mcpServer == nil {
		t.Fatal("MCP server not initialized")
	}

	// We can't easily test the actual tool execution without setting up a real server,
	// but we can verify that the client structure is properly initialized
	if client.baseURL == "" {
		t.Error("Base URL not set")
	}

	if client.httpClient == nil {
		t.Error("HTTP client not initialized")
	}
}
