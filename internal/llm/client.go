package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client interface {
	Chat(message, systemPrompt string) (string, error)
}

type OpenRouterClient struct {
	APIKey      string
	Model       string
	BaseURL     string
	Temperature float64
	MaxTokens   int
	httpClient  *http.Client
}

type OpenRouterRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
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
