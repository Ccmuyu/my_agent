package rag

import (
	"context"
	"testing"

	"github.com/Ccmuyu/my_agent/internal/config"
)

var (
	defaultChunkCfg  = config.ChunkConfig{Size: 512, Overlap: 50}
	defaultSearchCfg = config.SearchConfig{TopK: 5}
)

type mockVectorStore struct {
	vectors [][][]float64
	data    [][]map[string]any
	initErr error
}

func (m *mockVectorStore) Init(ctx context.Context) error {
	return m.initErr
}

func (m *mockVectorStore) Insert(ctx context.Context, vectors [][]float64, payloads []map[string]any) error {
	m.vectors = append(m.vectors, vectors)
	m.data = append(m.data, payloads)
	return nil
}

func (m *mockVectorStore) Search(ctx context.Context, query []float64, topK int) ([]VectorSearchResult, error) {
	if len(m.data) == 0 {
		return []VectorSearchResult{}, nil
	}
	lastBatch := m.data[len(m.data)-1]
	results := make([]VectorSearchResult, len(lastBatch))
	for i, payload := range lastBatch {
		results[i] = VectorSearchResult{
			ID:      "mock_" + string(rune(i)),
			Score:   0.9,
			Payload: payload,
		}
	}
	return results, nil
}

func (m *mockVectorStore) Delete(ctx context.Context, ids []string) error {
	return nil
}

func (m *mockVectorStore) Close() error {
	return nil
}

type mockEmbedder struct {
	vectors   [][]float64
	dimension int
	shouldErr bool
}

func (m *mockEmbedder) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if m.shouldErr {
		return nil, ErrMock
	}
	result := make([][]float64, len(texts))
	for i := range texts {
		vec := make([]float64, m.dimension)
		for j := range vec {
			vec[j] = float64(i+j) / float64(m.dimension)
		}
		result[i] = vec
	}
	return result, nil
}

func (m *mockEmbedder) GetDimension() int {
	return m.dimension
}

var ErrMock = &mockError{}

type mockError struct{}

func (e *mockError) Error() string {
	return "mock error"
}

func TestNewRAGService(t *testing.T) {
	store := &mockVectorStore{}
	embedder := &mockEmbedder{dimension: 768}

	svc, err := NewRAGService(store, embedder, defaultChunkCfg, defaultSearchCfg)
	if err != nil {
		t.Fatalf("NewRAGService() error = %v", err)
	}
	if svc == nil {
		t.Fatal("NewRAGService() returned nil service")
	}
}

func TestRAGService_IndexDocuments(t *testing.T) {
	store := &mockVectorStore{}
	embedder := &mockEmbedder{dimension: 768}

	svc, _ := NewRAGService(store, embedder, defaultChunkCfg, defaultSearchCfg)

	t.Run("index valid documents", func(t *testing.T) {
		err := svc.IndexDocuments(context.Background(), []string{"testdata/doc1.txt"})
		if err != nil {
			t.Errorf("IndexDocuments() error = %v", err)
		}
	})
}

func TestRAGService_Search(t *testing.T) {
	store := &mockVectorStore{}
	embedder := &mockEmbedder{dimension: 768}

	svc, _ := NewRAGService(store, embedder, defaultChunkCfg, defaultSearchCfg)

	t.Run("search with results", func(t *testing.T) {
		svc.IndexDocuments(context.Background(), []string{"testdata/doc1.txt"})
		result, err := svc.Search(context.Background(), "test query", 5)
		if err != nil {
			t.Errorf("Search() error = %v", err)
		}
		if result == "" {
			t.Error("Search() returned empty result")
		}
	})

	t.Run("search empty store", func(t *testing.T) {
		store.vectors = nil
		store.data = nil
		result, err := svc.Search(context.Background(), "test", 5)
		if err != nil {
			t.Errorf("Search() error = %v", err)
		}
		if result != "No relevant documents found." {
			t.Errorf("Search() = %q, want %q", result, "No relevant documents found.")
		}
	})
}

func TestRAGService_SearchWithSources(t *testing.T) {
	store := &mockVectorStore{}
	embedder := &mockEmbedder{dimension: 768}

	svc, _ := NewRAGService(store, embedder, defaultChunkCfg, defaultSearchCfg)
	svc.IndexDocuments(context.Background(), []string{"testdata/doc1.txt"})

	results, err := svc.SearchWithSources(context.Background(), "test", 5)
	if err != nil {
		t.Errorf("SearchWithSources() error = %v", err)
	}
	if len(results) == 0 {
		t.Error("SearchWithSources() returned empty results")
	}
}

func TestRAGService_ListDocuments(t *testing.T) {
	store := &mockVectorStore{}
	embedder := &mockEmbedder{dimension: 768}

	svc, _ := NewRAGService(store, embedder, defaultChunkCfg, defaultSearchCfg)

	t.Run("list empty", func(t *testing.T) {
		docs, err := svc.ListDocuments(context.Background())
		if err != nil {
			t.Errorf("ListDocuments() error = %v", err)
		}
		if len(docs) != 0 {
			t.Errorf("ListDocuments() len = %d, want 0", len(docs))
		}
	})

	t.Run("list after indexing", func(t *testing.T) {
		svc.IndexDocuments(context.Background(), []string{"testdata/doc1.txt"})
		docs, err := svc.ListDocuments(context.Background())
		if err != nil {
			t.Errorf("ListDocuments() error = %v", err)
		}
		if len(docs) == 0 {
			t.Error("ListDocuments() returned empty after indexing")
		}
	})
}

func TestRAGService_DeleteDocument(t *testing.T) {
	store := &mockVectorStore{}
	embedder := &mockEmbedder{dimension: 768}

	svc, _ := NewRAGService(store, embedder, defaultChunkCfg, defaultSearchCfg)
	svc.IndexDocuments(context.Background(), []string{"testdata/doc1.txt"})

	t.Run("delete existing doc", func(t *testing.T) {
		docs, _ := svc.ListDocuments(context.Background())
		if len(docs) > 0 {
			err := svc.DeleteDocument(context.Background(), docs[0].ID)
			if err != nil {
				t.Errorf("DeleteDocument() error = %v", err)
			}
		}
	})

	t.Run("delete non-existent doc", func(t *testing.T) {
		err := svc.DeleteDocument(context.Background(), "non_existent_id")
		if err == nil {
			t.Error("DeleteDocument() expected error for non-existent doc")
		}
	})
}

func TestRAGService_Close(t *testing.T) {
	store := &mockVectorStore{}
	embedder := &mockEmbedder{dimension: 768}

	svc, _ := NewRAGService(store, embedder, defaultChunkCfg, defaultSearchCfg)

	if err := svc.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}