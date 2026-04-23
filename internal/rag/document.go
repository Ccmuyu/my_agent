package rag

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"desktop-agent/internal/config"
)

type DocumentProcessor struct {
	embedder Embedder
	store    VectorStore
	chunkCfg config.ChunkConfig
	docIndex map[string][]string
}

func NewDocumentProcessor(
	embedder Embedder,
	store VectorStore,
	chunkCfg config.ChunkConfig,
) (*DocumentProcessor, error) {
	return &DocumentProcessor{
		embedder: embedder,
		store:    store,
		chunkCfg: chunkCfg,
		docIndex: make(map[string][]string),
	}, nil
}

func (p *DocumentProcessor) Process(ctx context.Context, paths []string) error {
	var allChunks []string
	var allPayloads []map[string]any

	for _, path := range paths {
		ext := strings.ToLower(filepath.Ext(path))

		var content string
		var err error

		switch ext {
		case ".md", ".txt", ".go", ".py", ".js", ".ts", ".java", ".c", ".cpp", ".h", ".json", ".yaml", ".yml", ".xml", ".html", ".css", ".sql", ".sh":
			content, err = p.readFile(path)
		case ".pdf":
			content, err = p.extractPDF(path)
		default:
			content, err = p.readFile(path)
		}

		if err != nil {
			return fmt.Errorf("failed to process file %s: %w", path, err)
		}

		chunks := ChunkText(content, p.chunkCfg.Size, p.chunkCfg.Overlap)

		docID := fmt.Sprintf("doc_%d", len(p.docIndex))
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

		p.docIndex[docID] = chunkIDs
	}

	if len(allChunks) == 0 {
		return nil
	}

	vectors, err := p.embedder.Embed(ctx, allChunks)
	if err != nil {
		return fmt.Errorf("failed to embed chunks: %w", err)
	}

	if err := p.store.Insert(ctx, vectors, allPayloads); err != nil {
		return fmt.Errorf("failed to insert vectors: %w", err)
	}

	return nil
}

func (p *DocumentProcessor) readFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (p *DocumentProcessor) extractPDF(path string) (string, error) {
	return "", fmt.Errorf("PDF extraction not implemented: use external tool like pdftotext")
}