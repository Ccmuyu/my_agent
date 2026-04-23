package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type Client interface {
	Chat(message, systemPrompt string) (string, error)
	ChatWithContext(ctx context.Context, message, systemPrompt string) (string, error)
	StreamChat(message, systemPrompt string) (<-chan string, error)
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type OpenRouterClient struct {
	APIKey      string
	Model       string
	BaseURL     string
	Temperature float64
	MaxTokens   int
	httpClient  HTTPClient
}

type OpenRouterRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
	Stream      bool      `json:"stream,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenRouterResponse struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message Message `json:"message"`
}

func NewOpenRouterClient(apiKey, model, baseURL string, temperature float64, maxTokens int) *OpenRouterClient {
	return &OpenRouterClient{
		APIKey:      apiKey,
		Model:       model,
		BaseURL:     baseURL,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (c *OpenRouterClient) Chat(message, systemPrompt string) (string, error) {
	messages := []Message{}
	if systemPrompt != "" {
		messages = append(messages, Message{Role: "system", Content: systemPrompt})
	}
	messages = append(messages, Message{Role: "user", Content: message})

	reqBody := OpenRouterRequest{
		Model:       c.Model,
		Messages:    messages,
		Temperature: c.Temperature,
		MaxTokens:   c.MaxTokens,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	log.Printf("[LLM] URL: %s, Model: %s", c.BaseURL, c.Model)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	log.Printf("[LLM] Response Status: %d, Body: %s", resp.StatusCode, string(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result OpenRouterResponse
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", err
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	return result.Choices[0].Message.Content, nil
}

func (c *OpenRouterClient) ChatWithContext(ctx context.Context, message, systemPrompt string) (string, error) {
	messages := []Message{}
	if systemPrompt != "" {
		messages = append(messages, Message{Role: "system", Content: systemPrompt})
	}
	messages = append(messages, Message{Role: "user", Content: message})

	reqBody := OpenRouterRequest{
		Model:       c.Model,
		Messages:    messages,
		Temperature: c.Temperature,
		MaxTokens:   c.MaxTokens,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result OpenRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	return result.Choices[0].Message.Content, nil
}

func (c *OpenRouterClient) StreamChat(message, systemPrompt string) (<-chan string, error) {
	messages := []Message{}
	if systemPrompt != "" {
		messages = append(messages, Message{Role: "system", Content: systemPrompt})
	}
	messages = append(messages, Message{Role: "user", Content: message})

	reqBody := OpenRouterRequest{
		Model:       c.Model,
		Messages:    messages,
		Temperature: c.Temperature,
		MaxTokens:   c.MaxTokens,
		Stream:      true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	ch := make(chan string, 100)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		reader := resp.Body
		buffer := make([]byte, 4096)
		lineBuffer := []byte{}

		for {
			n, err := reader.Read(buffer)
			if n > 0 {
				lineBuffer = append(lineBuffer, buffer[:n]...)
				lines := bytes.Split(lineBuffer, []byte("\n"))
				lineBuffer = lines[len(lines)-1]

				for i := 0; i < len(lines)-1; i++ {
					line := bytes.TrimSpace(lines[i])
					if len(line) == 0 || !bytes.HasPrefix(line, []byte("data: ")) {
						continue
					}
					if bytes.Equal(line, []byte("data: [DONE]")) {
						return
					}

					content := parseSSEData(line)
					if content != "" {
						select {
						case ch <- content:
						default:
						}
					}
				}
			}
			if err != nil {
				if err == io.EOF {
					break
				}
				return
			}
		}
	}()

	return ch, nil
}

func parseSSEData(line []byte) string {
	data := bytes.TrimPrefix(line, []byte("data: "))
	var chunk struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &chunk); err != nil {
		return ""
	}
	if len(chunk.Choices) > 0 {
		return chunk.Choices[0].Delta.Content
	}
	return ""
}
