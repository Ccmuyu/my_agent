package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Ccmuyu/my_agent/internal/config"
)

type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float64, error)
	GetDimension() int
}

type ollamaEmbedder struct {
	model   string
	baseURL string
	apiKey  string
	dim     int
}

type openAIEmbedder struct {
	model   string
	baseURL string
	apiKey  string
	dim     int
}

type ollamaEmbeddingsReq struct {
	Model   string   `json:"model"`
	Prompts []string `json:"prompts"`
}

type ollamaEmbeddingsResp struct {
	Embeddings [][]float64 `json:"embeddings"`
}

func (e *ollamaEmbedder) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	reqBody := ollamaEmbeddingsReq{
		Model:   e.model,
		Prompts: texts,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/api/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if e.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.apiKey)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned status: %d, body: %s", resp.StatusCode, string(body))
	}

	var result ollamaEmbeddingsResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Embeddings, nil
}

func (e *ollamaEmbedder) GetDimension() int {
	return e.dim
}

type openAIEmbeddingReq struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type openAIEmbeddingResp struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

func (e *openAIEmbedder) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	reqBody := openAIEmbeddingReq{
		Input: texts,
		Model: e.model,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/v1/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai returned status: %d, body: %s", resp.StatusCode, string(body))
	}

	var result openAIEmbeddingResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	embeddings := make([][]float64, len(result.Data))
	for i, d := range result.Data {
		embeddings[i] = d.Embedding
	}

	return embeddings, nil
}

func (e *openAIEmbedder) GetDimension() int {
	return e.dim
}

func NewEmbedder(cfg *config.EmbedderConfig) (Embedder, error) {
	switch strings.ToLower(cfg.Provider) {
	case "ollama":
		return newOllamaEmbedder(cfg)
	case "openai":
		return newOpenAIEmbedder(cfg)
	default:
		return newOllamaEmbedder(cfg)
	}
}

func newOllamaEmbedder(cfg *config.EmbedderConfig) (Embedder, error) {
	dim := 768
	if cfg.Model == "nomic-embed-text" {
		dim = 768
	} else if cfg.Model == "bge-m3" {
		dim = 1024
	}
	return &ollamaEmbedder{
		model:   cfg.Model,
		baseURL: cfg.BaseURL,
		apiKey:  cfg.APIKey,
		dim:     dim,
	}, nil
}

func newOpenAIEmbedder(cfg *config.EmbedderConfig) (Embedder, error) {
	dim := 1536
	if cfg.Model == "text-embedding-3-small" {
		dim = 1536
	} else if cfg.Model == "text-embedding-3-large" {
		dim = 3072
	}
	return &openAIEmbedder{
		model:   cfg.Model,
		baseURL: cfg.BaseURL,
		apiKey:  cfg.APIKey,
		dim:     dim,
	}, nil
}