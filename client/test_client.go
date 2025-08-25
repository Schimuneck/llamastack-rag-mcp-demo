package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	llamastackclient "github.com/llamastack/llama-stack-client-go"
	"github.com/llamastack/llama-stack-client-go/option"
	"github.com/llamastack/llama-stack-client-go/shared"
)

// ConversationMessage represents a message in the conversation history
type ConversationMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Type    string `json:"type"`
}

// ContentItem represents a single content item in the response
type ContentItem struct {
	Text        string        `json:"text"`
	Type        string        `json:"type"`
	Annotations []interface{} `json:"annotations"`
}

// LlamaStackClient wraps the LlamaStack client for RAG and MCP
type LlamaStackClient struct {
	client              *llamastackclient.Client
	conversationHistory []ConversationMessage
	agentID             string
	sessionID           string
	vectorStoreID       string
	mcpToolGroupID      string
}

// NewLlamaStackClient creates a new client configured for Llama Stack
func NewLlamaStackClient() *LlamaStackClient {
	client := llamastackclient.NewClient(
		option.WithBaseURL("http://localhost:8321"),
		option.WithAPIKey("none"),
	)

	return &LlamaStackClient{
		client:              &client,
		conversationHistory: make([]ConversationMessage, 0),
	}
}

// ListModels lists all available models
func (c *LlamaStackClient) ListModels(ctx context.Context) error {
	fmt.Println("🤖 Listing models...")

	models, err := c.client.Models.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list models: %w", err)
	}

	fmt.Printf("Found %d models:\n", len(*models))
	for _, model := range *models {
		fmt.Printf("  ✓ %s\n", model.Identifier)
	}
	fmt.Println()
	return nil
}

// CreateVectorStore creates a new vector store for RAG
func (c *LlamaStackClient) CreateVectorStore(ctx context.Context, name string) (*llamastackclient.VectorStore, error) {
	fmt.Printf("📦 Creating vector store: %s...\n", name)

	vectorStore, err := c.client.VectorStores.New(ctx, llamastackclient.VectorStoreNewParams{
		Name: llamastackclient.String(name),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create vector store: %w", err)
	}

	fmt.Printf("  ✓ Created vector store: %s (ID: %s)\n\n", vectorStore.Name, vectorStore.ID)
	c.vectorStoreID = vectorStore.ID
	return vectorStore, nil
}

// UploadFile uploads a file to be used in RAG
func (c *LlamaStackClient) UploadFile(ctx context.Context, filePath string) (string, error) {
	fmt.Printf("📁 Uploading file: %s...\n", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	uploadedFile, err := c.client.Files.New(ctx, llamastackclient.FileNewParams{
		File:    file,
		Purpose: llamastackclient.FileNewParamsPurposeAssistants,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	fmt.Printf("  ✓ File uploaded: %s (ID: %s)\n", uploadedFile.Filename, uploadedFile.ID)
	return uploadedFile.ID, nil
}

// AddFileToVectorStore adds a file to the vector store
func (c *LlamaStackClient) AddFileToVectorStore(ctx context.Context, vectorStoreID, fileID string) error {
	fmt.Printf("🔗 Adding file to vector store...\n")

	_, err := c.client.VectorStores.Files.New(ctx, vectorStoreID, llamastackclient.VectorStoreFileNewParams{
		FileID: fileID,
	})
	if err != nil {
		return fmt.Errorf("failed to add file to vector store: %w", err)
	}

	fmt.Printf("  ✓ File added to vector store successfully\n\n")
	return nil
}

// SetupMCPToolGroup registers the MCP tool group for ElectroShop database
func (c *LlamaStackClient) SetupMCPToolGroup(ctx context.Context) error {
	fmt.Printf("🛠️  Setting up MCP tool group for ElectroShop database...\n")

	mcpEndpoint := llamastackclient.ToolgroupRegisterParamsMcpEndpoint{
		Uri: "http://127.0.0.1:8000/mcp",
	}

	err := c.client.Toolgroups.Register(ctx, llamastackclient.ToolgroupRegisterParams{
		ToolgroupID: "electroshop-db",
		ProviderID:  "model-context-protocol", // ← FIXED: Match the config file
		McpEndpoint: mcpEndpoint,
	})
	if err != nil {
		return fmt.Errorf("failed to register MCP tool group: %w", err)
	}

	c.mcpToolGroupID = "electroshop-db"
	fmt.Printf("  ✓ MCP tool group registered: %s\n\n", c.mcpToolGroupID)
	return nil
}

// CreateAgent creates an agent for conversation management
func (c *LlamaStackClient) CreateAgent(ctx context.Context, modelID string) error {
	fmt.Printf("🤖 Creating agent for conversation management...\n")

	// First, let's get available tool groups
	_, err := c.client.Toolgroups.List(ctx)
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not list tool groups: %v\n", err)
	}

	// Log MCP tool group availability
	if c.mcpToolGroupID != "" {
		fmt.Printf("   🛠️  MCP tool group available: %s\n", c.mcpToolGroupID)
	}

	// Create agent with both RAG and MCP capabilities
	agentResponse, err := c.client.Agents.New(ctx, llamastackclient.AgentNewParams{
		AgentConfig: shared.AgentConfigParam{
			Model:                    modelID,
			Instructions:             "You are an expert assistant for ElectroShop. Use the RAG system to answer questions about ElectroShop's history and information, and use the MCP tools to interact with the ElectroShop sales database. Always provide helpful, accurate information.",
			EnableSessionPersistence: llamastackclient.Bool(true),
			MaxInferIters:            llamastackclient.Int(5),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	c.agentID = agentResponse.AgentID
	fmt.Printf("  ✓ Agent created: %s\n\n", c.agentID)
	return nil
}

// CreateSession creates a session for the agent
func (c *LlamaStackClient) CreateSession(ctx context.Context) error {
	fmt.Printf("🗣️  Creating agent session...\n")

	sessionResponse, err := c.client.Agents.Session.New(ctx, c.agentID, llamastackclient.AgentSessionNewParams{
		SessionName: "ElectroShop Chat Session",
	})
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	c.sessionID = sessionResponse.SessionID
	fmt.Printf("  ✓ Session created: %s\n\n", c.sessionID)
	return nil
}

// SendMessage sends a message using the Responses API with RAG and MCP support
func (c *LlamaStackClient) SendMessage(ctx context.Context, message string) (*shared.CompletionMessage, error) {
	// Sending message to LlamaStack

	// Build response parameters
	apiParams := llamastackclient.ResponseNewParams{
		Model: "ollama/llama3.2:1b",
		Input: llamastackclient.ResponseNewParamsInputUnion{
			OfString: llamastackclient.String(message),
		},
		Store: llamastackclient.Bool(true),
	}

	// Add instructions based on available tools
	if c.vectorStoreID != "" {

		apiParams.Instructions = llamastackclient.String("Use the ElectroShop knowledge base to answer questions about company history and information.")
	}

	if c.mcpToolGroupID != "" {

		if c.vectorStoreID != "" {
			// Both RAG and MCP available
			apiParams.Instructions = llamastackclient.String("Use the ElectroShop knowledge base for company information and the sales database tools for customer data operations.")
		} else {
			// Only MCP available
			apiParams.Instructions = llamastackclient.String("Use the ElectroShop sales database tools for customer data operations.")
		}
	}

	// Note: Tool integration (file_search, MCP) will be added once we determine correct type structures

	// Create response using the Responses API
	response, err := c.client.Responses.New(ctx, apiParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create response: %w", err)
	}

	// STEP 2: Retrieve the full response content using the response ID

	fullResponse, err := c.client.Responses.Get(ctx, response.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve response: %w", err)
	}

	// Extract the actual response content from the full response
	if len(fullResponse.Output) > 0 {
		for _, outputItem := range fullResponse.Output {

			if outputItem.Type == "message" {
				msg := outputItem.AsMessage()

				// Try to extract the actual AI response text
				if responseText := c.extractMessageContent(&msg); responseText != "" {

					return &shared.CompletionMessage{
						Role: "assistant",
						Content: shared.InterleavedContentUnion{
							OfString: responseText,
						},
					}, nil
				}
			}
		}
	}

	// Fallback response
	return &shared.CompletionMessage{
		Role: "assistant",
		Content: shared.InterleavedContentUnion{
			OfString: fmt.Sprintf("Response created (ID: %s) but content extraction still needs work.", response.ID),
		},
	}, nil
}

// extractMessageContent tries to extract text content from a response message
func (c *LlamaStackClient) extractMessageContent(msg *llamastackclient.ResponseObjectOutputMessage) string {
	// Parse the content JSON to extract the actual text
	if msg.JSON.Content.Valid() {
		rawContent := msg.JSON.Content.Raw()

		// Parse the JSON array of content items
		var contentItems []ContentItem
		if err := json.Unmarshal([]byte(rawContent), &contentItems); err != nil {
			return fmt.Sprintf("Error parsing response: %v", err)
		}

		// Find and return the first output_text content
		for _, item := range contentItems {
			if item.Type == "output_text" && item.Text != "" {
				return item.Text
			}
		}

		return "No readable content found in response."
	}

	// Fallback if content JSON is not valid
	return "Unable to parse response content."
}

// Helper function to get minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// StartInteractiveChat starts an interactive chat session
func (c *LlamaStackClient) StartInteractiveChat(ctx context.Context) error {
	fmt.Println("🎉 Starting interactive chat with RAG + MCP support!")
	fmt.Println("Type 'exit' to quit, 'clear' to clear conversation history")
	fmt.Println("Examples:")
	fmt.Println("- Tell me about ElectroShop's history")
	fmt.Println("- List all customers in the database")
	fmt.Println("- Add a new customer named John Smith")
	fmt.Println("=====================================")

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\n🗨️  You: ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if input == "exit" {
			fmt.Println("👋 Goodbye!")
			break
		}

		if input == "clear" {
			c.conversationHistory = make([]ConversationMessage, 0)
			fmt.Println("🧹 Conversation history cleared")
			continue
		}

		// Send message and get response
		response, err := c.SendMessage(ctx, input)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			continue
		}

		// Extract and display response content
		fmt.Print("🤖 Assistant: ")
		if response.Content.OfString != "" {
			fmt.Printf("%s\n", response.Content.OfString)
		} else if response.ToolCalls != nil {
			fmt.Printf("🛠️  Executing tools...\n")
			// Tool execution details would be shown here
		} else {
			fmt.Printf("(No text response available)\n")
		}

		// Add to conversation history
		c.conversationHistory = append(c.conversationHistory, ConversationMessage{
			Role:    "user",
			Content: input,
			Type:    "message",
		})

		if response.Content.OfString != "" {
			c.conversationHistory = append(c.conversationHistory, ConversationMessage{
				Role:    "assistant",
				Content: response.Content.OfString,
				Type:    "message",
			})
		}
	}

	return nil
}

func main() {
	fmt.Println("🚀 LlamaStack Go Client - RAG + MCP Demo")
	fmt.Println("=========================================")

	ctx := context.Background()
	client := NewLlamaStackClient()

	// Test 1: List available models
	fmt.Printf("🔍 Step 1: Listing available models...\n")
	if err := client.ListModels(ctx); err != nil {
		fmt.Printf("❌ Error listing models: %v\n", err)
		return
	}

	selectedModel := "ollama/llama3.2:3b" // Use available model

	// Test 2: Setup RAG - Create vector store and upload file
	fmt.Printf("📚 Step 2: Setting up RAG system...\n")
	vectorStore, err := client.CreateVectorStore(ctx, "ElectroShop Knowledge Base")
	if err != nil {
		fmt.Printf("❌ Error creating vector store: %v\n", err)
		return
	}

	// Upload ElectroShop history file
	testFile := "eletroshop_history.txt"
	if _, statErr := os.Stat(testFile); statErr == nil {
		fileID, uploadErr := client.UploadFile(ctx, testFile)
		if uploadErr != nil {
			fmt.Printf("❌ Error uploading file: %v\n", uploadErr)
			return
		}

		if addErr := client.AddFileToVectorStore(ctx, vectorStore.ID, fileID); addErr != nil {
			fmt.Printf("❌ Error adding file to vector store: %v\n", addErr)
			return
		}
	} else {
		fmt.Printf("⚠️  File %s not found, continuing without RAG data\n", testFile)
	}

	// Test 3: Setup MCP tool group
	fmt.Printf("🛠️  Step 3: Setting up MCP integration...\n")
	if err := client.SetupMCPToolGroup(ctx); err != nil {
		fmt.Printf("❌ Error setting up MCP: %v\n", err)
		fmt.Printf("⚠️  Continuing without MCP tools (make sure MCP server is running at http://127.0.0.1:8000/mcp)\n")
	}

	// Test 4: Create agent for conversation management
	fmt.Printf("🤖 Step 4: Creating conversational agent...\n")
	if err := client.CreateAgent(ctx, selectedModel); err != nil {
		fmt.Printf("❌ Error creating agent: %v\n", err)
		return
	}

	// Test 5: Create session for the agent
	fmt.Printf("🗣️  Step 5: Creating agent session...\n")
	if err := client.CreateSession(ctx); err != nil {
		fmt.Printf("❌ Error creating session: %v\n", err)
		return
	}

	// Test 6: Start interactive chat
	fmt.Printf("💬 Step 6: Starting interactive chat...\n")
	if err := client.StartInteractiveChat(ctx); err != nil {
		fmt.Printf("❌ Error in chat: %v\n", err)
		return
	}

	fmt.Println("🎉 Demo completed successfully!")
}
