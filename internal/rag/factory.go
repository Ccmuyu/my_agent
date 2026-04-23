package rag

import (
	"context"
	"fmt"

	"github.com/Ccmuyu/my_agent/internal/config"
)

func NewRAGServiceFromConfig(ctx context.Context, cfg *config.RAGConfig) (RAGService, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("RAG is not enabled")
	}

	embedder, err := NewEmbedder(&cfg.Embedder)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	vectorSize := embedder.GetDimension()

	store, err := NewQdrantClient(&cfg.VectorDB, vectorSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create vector store: %w", err)
	}

	if err := store.Init(ctx); err != nil {
		return nil, fmt.Errorf("failed to init vector store: %w", err)
	}

	return NewRAGService(store, embedder, cfg.Chunk, cfg.Search)
}