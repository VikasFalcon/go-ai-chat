package service

import "testing"

func TestChunkText_Basic(t *testing.T) {
	text := "one two three four five six seven eight nine ten eleven twelve thirteen fourteen fifteen"
	chunks := chunkText(text, 30, 10)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d: %v", len(chunks), chunks)
	}
	for _, c := range chunks {
		if len(c) == 0 {
			t.Fatalf("got an empty chunk in %v", chunks)
		}
	}
}

func TestChunkText_EmptyInput(t *testing.T) {
	if chunks := chunkText("   ", 100, 10); chunks != nil {
		t.Fatalf("expected nil for blank input, got %v", chunks)
	}
	if chunks := chunkText("", 100, 10); chunks != nil {
		t.Fatalf("expected nil for empty input, got %v", chunks)
	}
}

func TestChunkText_ShortTextSingleChunk(t *testing.T) {
	chunks := chunkText("just a short sentence", 800, 150)
	if len(chunks) != 1 {
		t.Fatalf("expected exactly 1 chunk for short text, got %d: %v", len(chunks), chunks)
	}
}

func TestChunkText_NormalizesWhitespace(t *testing.T) {
	chunks := chunkText("line one\n\n\nline   two\t\tline three", 800, 150)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0] != "line one line two line three" {
		t.Fatalf("whitespace not normalized, got: %q", chunks[0])
	}
}

func TestChunkText_BadOverlapFallsBackToDefault(t *testing.T) {
	// overlap >= chunkSize should not panic or infinite-loop; it should fall back.
	chunks := chunkText("a b c d e f g h i j k l m n o p q r s t", 20, 20)
	if len(chunks) == 0 {
		t.Fatalf("expected at least one chunk")
	}
}
