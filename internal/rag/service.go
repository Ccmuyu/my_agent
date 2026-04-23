package rag

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"desktop-agent/internal/config"
)

type RAGService interface {
	IndexDocuments(ctx context.Context, docPaths []string) error
	Search(ctx context.Context, query string, topK int) (string, error)
	SearchWithSources(ctx context.Context, query string, topK int) ([]SearchResult, error)
	ListDocuments(ctx context.Context) ([]DocumentInfo, error)
	DeleteDocument(ctx context.Context, docID string) error
	Close() error
}

type SearchResult struct {
	ID        string
	Content   string
	Score     float64
	FileName  string
	FilePath  string
	ChunkID   int
}

type DocumentInfo struct {
	ID       string
	FileName string
	FilePath string
	ChunkIDs []string
}

type ragService struct {
	store    VectorStore
	embedder Embedder
	chunkCfg config.ChunkConfig
	searchCfg config.SearchConfig
	docIndex map[string]*DocumentInfo
}

func NewRAGService(
	store VectorStore,
	embedder Embedder,
	chunkCfg config.ChunkConfig,
	searchCfg config.SearchConfig,
) (RAGService, error) {
	return &ragService{
		store:     store,
		embedder:  embedder,
		chunkCfg:  chunkCfg,
		searchCfg: searchCfg,
		docIndex:  make(map[string]*DocumentInfo),
	}, nil
}

func (s *ragService) IndexDocuments(ctx context.Context, docPaths []string) error {
	var allChunks []string
	var allPayloads []map[string]any

	for _, path := range docPaths {
		content, err := s.readFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		chunks := s.ChunkText(content)
		docID := fmt.Sprintf("doc_%d", len(s.docIndex))
		
		var chunkIDs []string

		for i, chunk := range chunks {
			chunkID := fmt.Sprintf("%s_chunk_%d", docID, i)
			chunkIDs = append(chunkIDs, chunkID)
			allChunks = append(allChunks, chunk)
			allPayloads = append(allPayloads, map[string]any{
				"document_id": docID,
				"chunk_id":    i,
				"file_name":   filepath.Base(path),
				"file_path":   path,
				"content":     chunk,
			})
		}

		s.docIndex[docID] = &DocumentInfo{
			ID:       docID,
			FileName: filepath.Base(path),
			FilePath: path,
			ChunkIDs: chunkIDs,
		}
	}

	if len(allChunks) == 0 {
		return nil
	}

	vectors, err := s.embedder.Embed(ctx, allChunks)
	if err != nil {
		return fmt.Errorf("failed to embed chunks: %w", err)
	}

	if err := s.store.Insert(ctx, vectors, allPayloads); err != nil {
		return fmt.Errorf("failed to insert vectors: %w", err)
	}

	return nil
}

func (s *ragService) Search(ctx context.Context, query string, topK int) (string, error) {
	if topK <= 0 {
		topK = s.searchCfg.TopK
	}

	queryVectors, err := s.embedder.Embed(ctx, []string{query})
	if err != nil {
		return "", fmt.Errorf("failed to embed query: %w", err)
	}

	results, err := s.store.Search(ctx, queryVectors[0], topK)
	if err != nil {
		return "", fmt.Errorf("failed to search: %w", err)
	}

	if len(results) == 0 {
		return "No relevant documents found.", nil
	}

	var context string
	for _, r := range results {
		content, ok := r.Payload["content"].(string)
		if ok {
			context += content + "\n\n"
		}
	}

	return context, nil
}

func (s *ragService) SearchWithSources(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	if topK <= 0 {
		topK = s.searchCfg.TopK
	}

	queryVectors, err := s.embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	results, err := s.store.Search(ctx, queryVectors[0], topK)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	if len(results) == 0 {
		return []SearchResult{}, nil
	}

	searchResults := make([]SearchResult, len(results))
	for i, r := range results {
		content, _ := r.Payload["content"].(string)
		fileName, _ := r.Payload["file_name"].(string)
		filePath, _ := r.Payload["file_path"].(string)
		chunkID, _ := r.Payload["chunk_id"].(float64)

		searchResults[i] = SearchResult{
			ID:       r.ID,
			Content:  content,
			Score:    r.Score,
			FileName: fileName,
			FilePath: filePath,
			ChunkID:  int(chunkID),
		}
	}

	return searchResults, nil
}

func (s *ragService) ListDocuments(ctx context.Context) ([]DocumentInfo, error) {
	docs := make([]DocumentInfo, 0, len(s.docIndex))
	for _, doc := range s.docIndex {
		docs = append(docs, *doc)
	}
	return docs, nil
}

func (s *ragService) DeleteDocument(ctx context.Context, docID string) error {
	doc, ok := s.docIndex[docID]
	if !ok {
		return fmt.Errorf("document not found: %s", docID)
	}

	if err := s.store.Delete(ctx, doc.ChunkIDs); err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	delete(s.docIndex, docID)
	return nil
}

func (s *ragService) Close() error {
	return s.store.Close()
}

func (s *ragService) readFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (s *ragService) ChunkText(text string) []string {
	if s.chunkCfg.Size <= 0 {
		s.chunkCfg.Size = 512
	}
	if s.chunkCfg.Overlap < 0 {
		s.chunkCfg.Overlap = 50
	}

	return ChunkText(text, s.chunkCfg.Size, s.chunkCfg.Overlap)
}

func ChunkText(text string, chunkSize, overlap int) []string {
	var chunks []string

	if chunkSize <= 0 {
		chunkSize = 512
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= chunkSize {
		overlap = chunkSize / 2
	}

	if len(text) <= chunkSize {
		return []string{text}
	}

	step := chunkSize - overlap
	for i := 0; i < len(text); i += step {
		end := i + chunkSize
		if end > len(text) {
			end = len(text)
		}
		chunks = append(chunks, text[i:end])
		
		if end == len(text) {
			break
		}
	}

	return chunks
}