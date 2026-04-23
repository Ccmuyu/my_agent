package tools

import (
	"context"
	"testing"

	"github.com/Ccmuyu/my_agent/internal/rag"
)

type mockRAGService struct {
	docs []rag.DocumentInfo
}

func (m *mockRAGService) IndexDocuments(ctx context.Context, docPaths []string) error {
	return nil
}

func (m *mockRAGService) Search(ctx context.Context, query string, topK int) (string, error) {
	return "mock search results", nil
}

func (m *mockRAGService) SearchWithSources(ctx context.Context, query string, topK int) ([]rag.SearchResult, error) {
	return []rag.SearchResult{
		{
			ID:       "test_1",
			Content:  "test content",
			Score:    0.9,
			FileName: "test.txt",
			FilePath: "/path/test.txt",
			ChunkID:  0,
		},
	}, nil
}

func (m *mockRAGService) ListDocuments(ctx context.Context) ([]rag.DocumentInfo, error) {
	return m.docs, nil
}

func (m *mockRAGService) DeleteDocument(ctx context.Context, docID string) error {
	for i, doc := range m.docs {
		if doc.ID == docID {
			m.docs = append(m.docs[:i], m.docs[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockRAGService) Close() error {
	return nil
}

func TestRAGTool_RegisterToRegistry(t *testing.T) {
	svc := &mockRAGService{}
	registry := NewToolRegistry()
	tool := NewRAGTool(svc, context.Background())

	tool.RegisterToRegistry(registry)

	tools := registry.ListTools()
	if len(tools) != 4 {
		t.Errorf("expected 4 tools registered, got %d", len(tools))
	}

	expectedTools := []string{"rag_search", "rag_add", "rag_list", "rag_delete"}
	for _, name := range expectedTools {
		found := false
		for _, t := range tools {
			if t.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected tool %s not found", name)
		}
	}
}

func TestRAGTool_search(t *testing.T) {
	svc := &mockRAGService{}
	tool := NewRAGTool(svc, context.Background())
	registry := NewToolRegistry()
	tool.RegisterToRegistry(registry)

	t.Run("search with query", func(t *testing.T) {
		result, err := registry.Call("rag_search", map[string]any{"query": "test"})
		if err != nil {
			t.Errorf("rag_search error = %v", err)
		}
		if result != "mock search results" {
			t.Errorf("rag_search result = %v, want %v", result, "mock search results")
		}
	})

	t.Run("search missing query", func(t *testing.T) {
		_, err := registry.Call("rag_search", map[string]any{})
		if err == nil {
			t.Error("rag_search expected error for missing query")
		}
	})

	t.Run("search with top_k", func(t *testing.T) {
		result, err := registry.Call("rag_search", map[string]any{"query": "test", "top_k": 3.0})
		if err != nil {
			t.Errorf("rag_search error = %v", err)
		}
		if result == nil {
			t.Error("rag_search returned nil result")
		}
	})
}

func TestRAGTool_addDocuments(t *testing.T) {
	svc := &mockRAGService{}
	tool := NewRAGTool(svc, context.Background())
	registry := NewToolRegistry()
	tool.RegisterToRegistry(registry)

	t.Run("add documents", func(t *testing.T) {
		result, err := registry.Call("rag_add", map[string]any{
			"paths": []any{"/path/to/doc1.txt", "/path/to/doc2.txt"},
		})
		if err != nil {
			t.Errorf("rag_add error = %v", err)
		}
		if result == nil {
			t.Error("rag_add returned nil")
		}
	})

	t.Run("add missing paths", func(t *testing.T) {
		_, err := registry.Call("rag_add", map[string]any{})
		if err == nil {
			t.Error("rag_add expected error for missing paths")
		}
	})

	t.Run("add empty paths", func(t *testing.T) {
		_, err := registry.Call("rag_add", map[string]any{"paths": []any{}})
		if err == nil {
			t.Error("rag_add expected error for empty paths")
		}
	})
}

func TestRAGTool_listDocuments(t *testing.T) {
	svc := &mockRAGService{
		docs: []rag.DocumentInfo{
			{ID: "doc_1", FileName: "test1.txt", FilePath: "/path/test1.txt"},
			{ID: "doc_2", FileName: "test2.md", FilePath: "/path/test2.md"},
		},
	}
	tool := NewRAGTool(svc, context.Background())
	registry := NewToolRegistry()
	tool.RegisterToRegistry(registry)

	t.Run("list documents", func(t *testing.T) {
		result, err := registry.Call("rag_list", map[string]any{})
		if err != nil {
			t.Errorf("rag_list error = %v", err)
		}
		resultStr, ok := result.(string)
		if !ok {
			t.Errorf("rag_list returned unexpected type %T", result)
		}
		if resultStr == "" {
			t.Error("rag_list returned empty string")
		}
	})

	t.Run("list empty", func(t *testing.T) {
		svc.docs = []rag.DocumentInfo{}
		result, err := registry.Call("rag_list", map[string]any{})
		if err != nil {
			t.Errorf("rag_list error = %v", err)
		}
		if result != "No documents indexed" {
			t.Errorf("rag_list = %v, want %v", result, "No documents indexed")
		}
	})
}

func TestRAGTool_deleteDocument(t *testing.T) {
	svc := &mockRAGService{
		docs: []rag.DocumentInfo{
			{ID: "doc_1", FileName: "test1.txt", FilePath: "/path/test1.txt"},
		},
	}
	tool := NewRAGTool(svc, context.Background())
	registry := NewToolRegistry()
	tool.RegisterToRegistry(registry)

	t.Run("delete document", func(t *testing.T) {
		result, err := registry.Call("rag_delete", map[string]any{"document_id": "doc_1"})
		if err != nil {
			t.Errorf("rag_delete error = %v", err)
		}
		if result == nil {
			t.Error("rag_delete returned nil")
		}
	})

	t.Run("delete missing document_id", func(t *testing.T) {
		_, err := registry.Call("rag_delete", map[string]any{})
		if err == nil {
			t.Error("rag_delete expected error for missing document_id")
		}
	})
}

func TestNewRAGTool(t *testing.T) {
	svc := &mockRAGService{}
	ctx := context.Background()

	tool := NewRAGTool(svc, ctx)
	if tool == nil {
		t.Error("NewRAGTool returned nil")
	}
	if tool.service != svc {
		t.Error("NewRAGTool did not set service")
	}
	if tool.ctx != ctx {
		t.Error("NewRAGTool did not set context")
	}
}