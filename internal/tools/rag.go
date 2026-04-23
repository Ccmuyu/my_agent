package tools

import (
	"context"
	"fmt"

	"desktop-agent/internal/rag"
)

type RAGTool struct {
	service rag.RAGService
	ctx     context.Context
}

func NewRAGTool(service rag.RAGService, ctx context.Context) *RAGTool {
	return &RAGTool{
		service: service,
		ctx:     ctx,
	}
}

func (t *RAGTool) RegisterToRegistry(r *ToolRegistry) {
	r.Register("rag_search", t.search, "搜索知识库，返回相关上下文", 1)
	r.Register("rag_add", t.addDocuments, "添加文档到知识库", 1)
	r.Register("rag_list", t.listDocuments, "列出已索引的文档", 1)
	r.Register("rag_delete", t.deleteDocument, "删除指定文档", 3)
}

func (t *RAGTool) search(params map[string]any) (any, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query is required")
	}

	topK := 5
	if k, ok := params["top_k"].(float64); ok {
		topK = int(k)
	}

	result, err := t.service.Search(t.ctx, query, topK)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	return result, nil
}

func (t *RAGTool) addDocuments(params map[string]any) (any, error) {
	paths, ok := params["paths"].([]any)
	if !ok || len(paths) == 0 {
		return nil, fmt.Errorf("paths is required and must be non-empty")
	}

	docPaths := make([]string, len(paths))
	for i, p := range paths {
		path, ok := p.(string)
		if !ok {
			return nil, fmt.Errorf("path must be a string")
		}
		docPaths[i] = path
	}

	err := t.service.IndexDocuments(t.ctx, docPaths)
	if err != nil {
		return nil, fmt.Errorf("failed to index documents: %w", err)
	}

	return fmt.Sprintf("Successfully indexed %d documents", len(docPaths)), nil
}

func (t *RAGTool) listDocuments(params map[string]any) (any, error) {
	docs, err := t.service.ListDocuments(t.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}

	if len(docs) == 0 {
		return "No documents indexed", nil
	}

	result := "Indexed documents:\n"
	for _, doc := range docs {
		result += fmt.Sprintf("- %s (%s)\n", doc.FileName, doc.FilePath)
	}

	return result, nil
}

func (t *RAGTool) deleteDocument(params map[string]any) (any, error) {
	docID, ok := params["document_id"].(string)
	if !ok || docID == "" {
		return nil, fmt.Errorf("document_id is required")
	}

	err := t.service.DeleteDocument(t.ctx, docID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete document: %w", err)
	}

	return fmt.Sprintf("Document %s deleted", docID), nil
}