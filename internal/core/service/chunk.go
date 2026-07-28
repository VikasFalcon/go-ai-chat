package service

import "strings"

// chunkText splits text into overlapping, word-boundary-aligned chunks of
// approximately chunkSize runes, sliding forward by (chunkSize - overlap)
// each step. Overlap keeps context from being cut across a chunk boundary,
// which improves retrieval quality for questions that span two chunks.
//
// Whitespace (including newlines from PDF extraction) is normalized to
// single spaces first, since layout whitespace carries no semantic meaning
// once we're chunking for embeddings.
func chunkText(text string, chunkSize, overlap int) []string {
	text = normalizeWhitespace(text)
	if text == "" {
		return nil
	}
	if chunkSize <= 0 {
		chunkSize = 800
	}
	if overlap < 0 || overlap >= chunkSize {
		overlap = chunkSize / 5
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var chunks []string
	var cur strings.Builder
	var wordBuf []string

	flush := func() {
		if cur.Len() > 0 {
			chunks = append(chunks, strings.TrimSpace(cur.String()))
		}
	}

	for _, w := range words {
		if cur.Len()+len(w)+1 > chunkSize && cur.Len() > 0 {
			flush()
			// start next chunk with the overlap tail of the previous one
			overlapWords := wordsWithinLen(wordBuf, overlap)
			cur.Reset()
			for _, ow := range overlapWords {
				cur.WriteString(ow)
				cur.WriteByte(' ')
			}
			wordBuf = append([]string{}, overlapWords...)
		}
		cur.WriteString(w)
		cur.WriteByte(' ')
		wordBuf = append(wordBuf, w)
	}
	flush()

	return chunks
}

// wordsWithinLen returns the trailing words of ws whose combined length
// (with single-space separators) is <= maxLen.
func wordsWithinLen(ws []string, maxLen int) []string {
	total := 0
	start := len(ws)
	for i := len(ws) - 1; i >= 0; i-- {
		total += len(ws[i]) + 1
		if total > maxLen {
			break
		}
		start = i
	}
	return ws[start:]
}

func normalizeWhitespace(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}
