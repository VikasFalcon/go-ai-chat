package service

import (
	"context"
	"strings"
	"testing"

	"github.com/VikasFalcon/go-ai-chat/internal/core/domain"
)

// --- fakes implementing the outbound ports -------------------------------

// fakeEmbedder produces a crude bag-of-words embedding so that semantically
// similar text (sharing more words) ends up with a higher cosine similarity.
// This is good enough to test threshold gating without needing real Ollama.
type fakeEmbedder struct{ vocab map[string]int }

func newFakeEmbedder() *fakeEmbedder { return &fakeEmbedder{vocab: map[string]int{}} }

func (f *fakeEmbedder) Embed(_ context.Context, text string) ([]float64, error) {
	words := strings.Fields(strings.ToLower(text))
	for _, w := range words {
		if _, ok := f.vocab[w]; !ok {
			f.vocab[w] = len(f.vocab)
		}
	}
	vec := make([]float64, len(f.vocab)+64) // headroom for words seen later
	for _, w := range words {
		vec[f.vocab[w]]++
	}
	return vec, nil
}

type fakeGenerator struct{}

func (fakeGenerator) Generate(_ context.Context, prompt string) (string, error) {
	return "ANSWER_FOR: " + prompt, nil
}

type fakePrompts struct{}

func (fakePrompts) Build(chunks, question string) (string, error) {
	return "CTX[" + chunks + "]Q[" + question + "]", nil
}

type fakeRepo struct{ docs []domain.Document }

func (r *fakeRepo) Add(_ context.Context, d domain.Document) error {
	d.ID = len(r.docs) + 1
	r.docs = append(r.docs, d)
	return nil
}
func (r *fakeRepo) Count() int { return len(r.docs) }
func (r *fakeRepo) HasSource(_ context.Context, source string) (bool, error) {
	for _, d := range r.docs {
		if d.Source == source {
			return true, nil
		}
	}
	return false, nil
}
func (r *fakeRepo) Search(_ context.Context, query []float64, topK int) ([]domain.ScoredDocument, error) {
	out := make([]domain.ScoredDocument, 0, len(r.docs))
	for _, d := range r.docs {
		score := cosine(query, d.Embedding)
		out = append(out, domain.ScoredDocument{Document: d, Score: score})
	}
	// simple selection sort by score desc (test-only, fine for small N)
	for i := range out {
		best := i
		for j := i + 1; j < len(out); j++ {
			if out[j].Score > out[best].Score {
				best = j
			}
		}
		out[i], out[best] = out[best], out[i]
	}
	if topK > len(out) {
		topK = len(out)
	}
	return out[:topK], nil
}

func TestRAGService_IngestPDF_IsIdempotentBySource(t *testing.T) {
	repo := &fakeRepo{}
	rag := NewRAGService(newFakeEmbedder(), repo, fakeGenerator{}, fakePrompts{}, fakeLoader{text: "one two three"}, 3, 0.5, 800, 150)

	first, err := rag.IngestPDF(context.Background(), "handbook.pdf")
	if err != nil {
		t.Fatalf("first ingestion failed: %v", err)
	}
	second, err := rag.IngestPDF(context.Background(), "handbook.pdf")
	if err != nil {
		t.Fatalf("second ingestion failed: %v", err)
	}
	if first != 1 || second != 0 || repo.Count() != 1 {
		t.Fatalf("first=%d second=%d count=%d; want 1, 0, 1", first, second, repo.Count())
	}
}

func cosine(a, b []float64) float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	var dot, magA, magB float64
	for i := 0; i < n; i++ {
		dot += a[i] * b[i]
	}
	for _, v := range a {
		magA += v * v
	}
	for _, v := range b {
		magB += v * v
	}
	if magA == 0 || magB == 0 {
		return 0
	}
	return dot / (sqrt(magA) * sqrt(magB))
}

func sqrt(x float64) float64 {
	if x == 0 {
		return 0
	}
	z := x
	for i := 0; i < 50; i++ {
		z -= (z*z - x) / (2 * z)
	}
	return z
}

type fakeLoader struct{ text string }

func (f fakeLoader) Load(_ string) (string, error) { return f.text, nil }

// --- the actual test -------------------------------------------------------

func TestRAGService_TopicGating_EndToEnd(t *testing.T) {
	pdfText := `NPCI Digital Rupee CBDC Platform Overview.
	The Central Bank Digital Currency CBDC platform is built using Go Apache Kafka
	and Hyperledger Fabric. It enables secure and scalable digital rupee transactions.
	Kafka is used as the event streaming backbone between core banking services and
	the Hyperledger Fabric ledger network.`

	embedder := newFakeEmbedder()
	repo := &fakeRepo{}
	rag := NewRAGService(embedder, repo, fakeGenerator{}, fakePrompts{}, fakeLoader{text: pdfText},
		3,   // topK
		0.3, // similarity threshold
		120, // chunk size (small, forces multiple chunks)
		20,  // overlap
	)

	ctx := context.Background()

	n, err := rag.IngestPDF(ctx, "handbook.pdf")
	if err != nil {
		t.Fatalf("IngestPDF failed: %v", err)
	}
	if n < 2 {
		t.Fatalf("expected multiple chunks from a ~350-char doc with chunkSize=120, got %d", n)
	}
	if rag.DocumentCount() != n {
		t.Fatalf("DocumentCount() = %d, want %d", rag.DocumentCount(), n)
	}

	// On-topic question should get a real generated answer, not the off-topic fallback.
	onTopic, err := rag.Chat(ctx, "What technology does the CBDC platform use for the ledger?")
	if err != nil {
		t.Fatalf("Chat (on-topic) failed: %v", err)
	}
	if onTopic == domain.OffTopicAnswer {
		t.Fatalf("expected a real answer for an on-topic question, got the off-topic fallback")
	}
	if !strings.HasPrefix(onTopic, "ANSWER_FOR:") {
		t.Fatalf("expected generator output, got: %q", onTopic)
	}

	// Off-topic question (no vocabulary overlap with the ingested doc) should
	// be rejected before ever reaching the generator.
	offTopic, err := rag.Chat(ctx, "banana smoothie recipe ingredients blender")
	if err != nil {
		t.Fatalf("Chat (off-topic) failed: %v", err)
	}
	if offTopic != domain.OffTopicAnswer {
		t.Fatalf("expected off-topic fallback, got: %q", offTopic)
	}

	// Empty question is a hard validation error, not a topic-gating decision.
	if _, err := rag.Chat(ctx, "   "); err != domain.ErrEmptyInput {
		t.Fatalf("expected ErrEmptyInput for blank question, got: %v", err)
	}

	// Ask() must behave identically to Chat() (kept as an alias). Reuse the
	// same on-topic question validated above.
	aliasAnswer, err := rag.Ask(ctx, "What technology does the CBDC platform use for the ledger?")
	if err != nil {
		t.Fatalf("Ask failed: %v", err)
	}
	if aliasAnswer == domain.OffTopicAnswer {
		t.Fatalf("Ask() unexpectedly returned off-topic fallback for an on-topic question")
	}
}

func TestRAGService_IngestPDF_RejectsNonPDF(t *testing.T) {
	rag := NewRAGService(newFakeEmbedder(), &fakeRepo{}, fakeGenerator{}, fakePrompts{}, fakeLoader{text: "x"}, 3, 0.5, 800, 150)
	if _, err := rag.IngestPDF(context.Background(), "notes.txt"); err != domain.ErrUnsupportedFileType {
		t.Fatalf("expected ErrUnsupportedFileType, got: %v", err)
	}
}
