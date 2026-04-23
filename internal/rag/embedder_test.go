package rag

import (
	"testing"

	"desktop-agent/internal/config"
)

func TestEmbedderFactory(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		model    string
		wantDim  int
		wantErr  bool
	}{
		{
			name:     "ollama default",
			provider: "ollama",
			model:    "nomic-embed-text",
			wantDim:  768,
			wantErr:  false,
		},
		{
			name:     "ollama bge-m3",
			provider: "ollama",
			model:    "bge-m3",
			wantDim:  1024,
			wantErr:  false,
		},
		{
			name:     "openai text-embedding-3-small",
			provider: "openai",
			model:    "text-embedding-3-small",
			wantDim:  1536,
			wantErr:  false,
		},
		{
			name:     "openai text-embedding-3-large",
			provider: "openai",
			model:    "text-embedding-3-large",
			wantDim:  3072,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.EmbedderConfig{
				Provider: tt.provider,
				Model:    tt.model,
				BaseURL:  "http://localhost:11434",
			}
			e, err := NewEmbedder(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEmbedder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && e.GetDimension() != tt.wantDim {
				t.Errorf("NewEmbedder() dim = %v, want %v", e.GetDimension(), tt.wantDim)
			}
		})
	}
}