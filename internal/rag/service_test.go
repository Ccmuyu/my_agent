package rag

import (
	"testing"
)

func TestChunkText(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		chunkSize int
		overlap   int
		wantLen   int
	}{
		{
			name:      "text shorter than chunk size",
			text:      "hello world",
			chunkSize: 512,
			overlap:   50,
			wantLen:   1,
		},
		{
			name:      "text exactly chunk size",
			text:      "a",
			chunkSize: 1,
			overlap:   0,
			wantLen:   1,
		},
		{
			name:      "text larger than chunk size",
			text:      "abcdefghijklmnopqrstuvwxyz",
			chunkSize: 10,
			overlap:   2,
			wantLen:   3,
		},
		{
			name:      "zero overlap",
			text:      "abcdefghij",
			chunkSize: 5,
			overlap:   0,
			wantLen:   2,
		},
		{
			name:      "overlap larger than chunk size",
			text:      "abcdefghij",
			chunkSize: 5,
			overlap:   10,
			wantLen:   3,
		},
		{
			name:      "empty text",
			text:      "",
			chunkSize: 512,
			overlap:   50,
			wantLen:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ChunkText(tt.text, tt.chunkSize, tt.overlap)
			if len(got) != tt.wantLen {
				t.Errorf("ChunkText() len = %v, want %v", len(got), tt.wantLen)
			}
		})
	}
}