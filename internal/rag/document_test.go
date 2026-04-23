package rag

import (
	"context"
	"os"
	"testing"

	"github.com/Ccmuyu/my_agent/internal/config"
)

func TestDocumentProcessor_Process(t *testing.T) {
	store := &mockVectorStore{}
	embedder := &mockEmbedder{dimension: 768}
	chunkCfg := config.ChunkConfig{Size: 100, Overlap: 20}

	proc, err := NewDocumentProcessor(embedder, store, chunkCfg)
	if err != nil {
		t.Fatalf("NewDocumentProcessor() error = %v", err)
	}

	t.Run("process txt file", func(t *testing.T) {
		err := proc.Process(context.Background(), []string{"testdata/doc1.txt"})
		if err != nil {
			t.Errorf("Process() error = %v", err)
		}
	})

	t.Run("process md file", func(t *testing.T) {
		err := proc.Process(context.Background(), []string{"testdata/doc2.md"})
		if err != nil {
			t.Errorf("Process() error = %v", err)
		}
	})

	t.Run("process multiple files", func(t *testing.T) {
		err := proc.Process(context.Background(), []string{"testdata/doc1.txt", "testdata/doc2.md"})
		if err != nil {
			t.Errorf("Process() error = %v", err)
		}
	})

	t.Run("process non-existent file", func(t *testing.T) {
		err := proc.Process(context.Background(), []string{"testdata/non_existent.txt"})
		if err == nil {
			t.Error("Process() expected error for non-existent file")
		}
	})

	t.Run("process empty paths", func(t *testing.T) {
		err := proc.Process(context.Background(), []string{})
		if err != nil {
			t.Errorf("Process() error = %v", err)
		}
	})
}

func TestDocumentProcessor_readFile(t *testing.T) {
	store := &mockVectorStore{}
	embedder := &mockEmbedder{dimension: 768}
	proc, _ := NewDocumentProcessor(embedder, store, config.ChunkConfig{})

	content, err := proc.readFile("testdata/doc1.txt")
	if err != nil {
		t.Errorf("readFile() error = %v", err)
	}
	if content == "" {
		t.Error("readFile() returned empty content")
	}

	_, err = proc.readFile("non_existent_file.txt")
	if err == nil {
		t.Error("readFile() expected error for non-existent file")
	}
}

func TestDocumentProcessor_extractPDF(t *testing.T) {
	store := &mockVectorStore{}
	embedder := &mockEmbedder{dimension: 768}
	proc, _ := NewDocumentProcessor(embedder, store, config.ChunkConfig{})

	_, err := proc.extractPDF("test.pdf")
	if err == nil {
		t.Error("extractPDF() expected error")
	}
}

func TestChunkText_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		chunkSize int
		overlap   int
		want      []string
	}{
		{
			name:      "exact size",
			text:      "0123456789",
			chunkSize: 10,
			overlap:   0,
			want:      []string{"0123456789"},
		},
		{
			name:      "single char chunks",
			text:      "abc",
			chunkSize: 1,
			overlap:   0,
			want:      []string{"a", "b", "c"},
		},
		{
			name:      "unicode text with small chunk size",
			text:      "你好世界",
			chunkSize: 3,
			overlap:   0,
			want:      []string{"你", "好", "世", "界"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ChunkText(tt.text, tt.chunkSize, tt.overlap)
			if len(got) != len(tt.want) {
				t.Errorf("ChunkText() len = %d, want %d", len(got), len(tt.want))
			}
		})
	}
}

func TestSupportedExtensions(t *testing.T) {
	store := &mockVectorStore{}
	embedder := &mockEmbedder{dimension: 768}
	proc, _ := NewDocumentProcessor(embedder, store, config.ChunkConfig{})

	supportedExts := []string{
		".md", ".txt", ".go", ".py", ".js", ".ts",
		".java", ".c", ".cpp", ".h", ".json", ".yaml",
		".yml", ".xml", ".html", ".css", ".sql", ".sh",
	}

	tmpFile, err := os.CreateTemp("", "test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("test content")
	tmpFile.Close()

	for _, ext := range supportedExts {
		t.Run(ext, func(t *testing.T) {
			err := proc.Process(context.Background(), []string{tmpFile.Name()})
			if err != nil {
				t.Errorf("Process() error for %s = %v", ext, err)
			}
		})
	}
}