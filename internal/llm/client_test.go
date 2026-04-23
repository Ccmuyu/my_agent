package llm

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type mockHTTPClient struct {
	response   string
	err        error
	statusCode int
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	statusCode := m.statusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(m.response)),
	}, nil
}

func TestNewOpenRouterClient(t *testing.T) {
	client := NewOpenRouterClient("test-key", "test-model", "https://api.test.com", 0.7, 1024)

	if client.APIKey != "test-key" {
		t.Errorf("APIKey = %v, want %v", client.APIKey, "test-key")
	}
	if client.Model != "test-model" {
		t.Errorf("Model = %v, want %v", client.Model, "test-model")
	}
	if client.BaseURL != "https://api.test.com" {
		t.Errorf("BaseURL = %v, want %v", client.BaseURL, "https://api.test.com")
	}
	if client.Temperature != 0.7 {
		t.Errorf("Temperature = %v, want %v", client.Temperature, 0.7)
	}
	if client.MaxTokens != 1024 {
		t.Errorf("MaxTokens = %v, want %v", client.MaxTokens, 1024)
	}
}

func TestOpenRouterClient_Chat(t *testing.T) {
	client := &OpenRouterClient{
		APIKey:      "noop",
		Model:       "deepseek-r1-0528",
		BaseURL:     "https://api.llm7.io/v1",
		Temperature: 0.7,
		MaxTokens:   100,
		httpClient: &mockHTTPClient{
			response: `{"choices":[{"message":{"content":"test response"}}]}`,
		},
	}

	t.Run("chat with system prompt", func(t *testing.T) {
		resp, err := client.Chat("hello", "you are a helpful assistant")
		if err != nil {
			t.Errorf("Chat() error = %v", err)
		}
		if resp == "" {
			t.Error("Chat() returned empty response")
		}
	})

	t.Run("chat without system prompt", func(t *testing.T) {
		resp, err := client.Chat("hello", "")
		if err != nil {
			t.Errorf("Chat() error = %v", err)
		}
		if resp == "" {
			t.Error("Chat() returned empty response")
		}
	})
}

func TestParseSSEData(t *testing.T) {
	tests := []struct {
		name     string
		line     []byte
		want     string
	}{
		{
			name: "valid delta",
			line: []byte(`data: {"choices":[{"delta":{"content":"Hello"}}]}`),
			want: "Hello",
		},
		{
			name: "empty content",
			line: []byte(`data: {"choices":[{"delta":{"content":""}}]}`),
			want: "",
		},
		{
			name:     "invalid json",
			line:     []byte(`data: invalid json`),
			want:     "",
		},
		{
			name:     "done marker",
			line:     []byte(`data: [DONE]`),
			want:     "",
		},
		{
			name:     "empty line",
			line:     []byte(``),
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSSEData(tt.line)
			if got != tt.want {
				t.Errorf("parseSSEData() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStreamChat(t *testing.T) {
	t.Run("stream interface", func(t *testing.T) {
		client := NewOpenRouterClient("noop", "test-model", "https://api.llm7.io/v1", 0.7, 100)
		
		streamCh, err := client.StreamChat("test", "")
		if err != nil {
			t.Skip("Skipping streaming test - API may not be available")
		}
		if streamCh == nil {
			t.Error("StreamChat() returned nil channel")
		}
		
		go func() {
			for range streamCh {
			}
		}()
	})
}

func TestMessage_Structure(t *testing.T) {
	msg := Message{
		Role:    "user",
		Content: "test content",
	}

	if msg.Role != "user" {
		t.Errorf("Role = %v, want %v", msg.Role, "user")
	}
	if msg.Content != "test content" {
		t.Errorf("Content = %v, want %v", msg.Content, "test content")
	}
}

func TestOpenRouterRequest_JSON(t *testing.T) {
	req := OpenRouterRequest{
		Model:       "test-model",
		Messages:    []Message{{Role: "user", Content: "hello"}},
		Temperature: 0.7,
		MaxTokens:   1024,
		Stream:      true,
	}

	if req.Model != "test-model" {
		t.Errorf("Model = %v, want %v", req.Model, "test-model")
	}
	if !req.Stream {
		t.Error("Stream should be true")
	}
}

func TestContextCancellation(t *testing.T) {
	client := NewOpenRouterClient("test", "model", "http://localhost:9999", 0.7, 100)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.ChatWithContext(ctx, "test", "")
	if err == nil {
		t.Error("ChatWithContext() expected error for cancelled context")
	}
}

func TestChatWithContext_Success(t *testing.T) {
	client := &OpenRouterClient{
		APIKey:      "noop",
		Model:       "test-model",
		BaseURL:     "https://api.test.com/v1",
		Temperature: 0.7,
		MaxTokens:   100,
		httpClient: &mockHTTPClient{
			response: `{"choices":[{"message":{"content":"success response"}}]}`,
		},
	}

	resp, err := client.ChatWithContext(context.Background(), "hello", "system prompt")
	if err != nil {
		t.Errorf("ChatWithContext() error = %v", err)
	}
	if resp != "success response" {
		t.Errorf("ChatWithContext() = %v, want %v", resp, "success response")
	}
}

func TestChatWithContext_InvalidStatusCode(t *testing.T) {
	client := &OpenRouterClient{
		APIKey:      "noop",
		Model:       "test-model",
		BaseURL:     "https://api.test.com/v1",
		Temperature: 0.7,
		MaxTokens:   100,
		httpClient: &mockHTTPClient{
			response:   `{"error":"invalid request"}`,
			statusCode: http.StatusBadRequest,
		},
	}

	_, err := client.Chat("test", "")
	if err == nil {
		t.Error("Chat() expected error for status 400")
	}
}

func TestChat_EmptyChoices(t *testing.T) {
	client := &OpenRouterClient{
		APIKey:      "noop",
		Model:       "test-model",
		BaseURL:     "https://api.test.com/v1",
		Temperature: 0.7,
		MaxTokens:   100,
		httpClient: &mockHTTPClient{
			response: `{"choices":[]}`,
		},
	}

	_, err := client.Chat("test", "")
	if err == nil {
		t.Error("Chat() expected error for empty choices")
	}
	if err != nil && err.Error() != "no response from API" {
		t.Errorf("Chat() error = %v, want 'no response from API'", err)
	}
}