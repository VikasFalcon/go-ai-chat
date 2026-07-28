# go-ai-chat — PDF RAG Chat Service (Local Ollama)

A Go-based RAG (Retrieval-Augmented Generation) API server that:

1. Loads PDF file(s) into the server (on startup, and/or via an upload endpoint).
2. Extracts, chunks, and embeds the PDF text using a local **Ollama** embedding model.
3. Stores the chunks + embeddings in a small embedded **vector store** (JSON-persisted, no external DB required).
4. Exposes a `POST /api/chat` endpoint: if your question is relevant to the ingested PDF content, it answers using RAG over your document; otherwise it replies with the literal string:

   ```
   Question is not related to topic
   ```

Built as an extension of the existing hexagonal-architecture codebase (`cmd/server`, `internal/core`, `internal/adapter`) — no rewrite, just new adapters and a gated service method.

---

## 1. What's new in this version

| Area | Change |
|---|---|
| `internal/adapter/outbound/pdfloader/` | **New.** Extracts plain text from a PDF (pure Go, via `github.com/ledongthuc/pdf` — no poppler/cgo needed). |
| `internal/core/service/chunk.go` | **New.** Word-boundary-aligned sliding-window chunker with overlap. |
| `internal/adapter/outbound/vectorstore/` | **Renamed** from `memory/`. Same in-memory cosine-similarity store, now with JSON disk persistence (survives restarts) and returns *scored* results. |
| `internal/core/service/rag.go` | `Chat()` now does topic-gating: below `SIMILARITY_THRESHOLD`, it returns `"Question is not related to topic"` **without calling the LLM**. Added `IngestPDF()`. |
| `internal/adapter/inbound/http/` | Added `POST /api/ingest` (multipart PDF upload) and `GET /api/stats`. |
| `internal/config/config.go` | New env vars: `SIMILARITY_THRESHOLD`, `CHUNK_SIZE`, `CHUNK_OVERLAP`, `PDF_DIR`, `VECTOR_STORE_PATH`, `MAX_UPLOAD_SIZE_MB`. |
| `cmd/server/main.go` | On startup, auto-ingests every `*.pdf` found in `PDF_DIR` (default `./data/pdfs`), and loads/persists the vector store. |

`go.mod` / `go.sum` / `vendor/` were updated to include `github.com/ledongthuc/pdf` (vendored, so `go build` works fully offline).

---

## 2. Architecture (unchanged shape, new pieces marked with a star)

```
cmd/server/main.go                 wires everything together, auto-ingests PDF_DIR on boot

internal/core/
  domain/        Document, ScoredDocument*, sentinel errors, OffTopicAnswer*
  port/          ChatService (+ IngestPDF, DocumentCount*), DocumentRepository, DocumentLoader*
  service/       RAGService: Chat (topic-gated*), Ask (alias), Ingest, IngestPDF*, chunkText*

internal/adapter/
  inbound/http/      Gin handlers + router: /api/chat /api/ask /api/ingest* /api/stats* /api/health
  outbound/ollama/    Ollama /api/generate + /api/embed client (unchanged)
  outbound/prompt/    text/template prompt builder (unchanged)
  outbound/vectorstore/  * renamed from memory/, adds JSON persistence + scored search
  outbound/pdfloader/    * new: PDF -> plain text
```

The request flow for `/api/chat`:

```
question --> Embed (Ollama) --> Search vector store (top-K, scored)
                                        |
                         best score < SIMILARITY_THRESHOLD ?
                              yes                    no
                 "Question is not related       Build RAG prompt from
                      to topic"                 top chunks --> Generate
                    (no LLM call)                (Ollama) --> answer
```

---

## 3. Prerequisites

- **Go 1.25+** (matches your existing `go.mod`; this repo vendors all dependencies so no network access is needed to build)
- **[Ollama](https://ollama.com)** installed and running locally
- Two models pulled in Ollama:
  ```bash
  ollama pull nomic-embed-text
  ollama pull qwen3:8b
  ```
  (Or any chat/embedding models you prefer — see env vars below.)

---

## 4. Setup

```bash
# from the project root
mkdir -p data/pdfs

# put the PDF(s) you want the server to load on startup here:
cp /path/to/your/handbook.pdf data/pdfs/

# make sure Ollama is running
ollama serve &          # if not already running as a service
```

### Environment variables (all optional — sensible defaults shown)

| Var | Default | Meaning |
|---|---|---|
| `PORT` | `8080` | HTTP port |
| `OLLAMA_URL` | `http://localhost:11434` | Ollama base URL |
| `CHAT_MODEL` | `qwen3:8b` | Ollama model used for generation |
| `EMBED_MODEL` | `nomic-embed-text` | Ollama model used for embeddings |
| `OLLAMA_TIMEOUT` | `60s` | HTTP client timeout for Ollama calls |
| `RAG_TOP_K` | `3` | How many chunks to retrieve per question |
| `SIMILARITY_THRESHOLD` | `0.5` | Minimum cosine similarity to treat a question as "on topic". Lower it if valid questions are being rejected; raise it if off-topic questions get answered. |
| `CHUNK_SIZE` | `800` | Approx. characters per chunk before embedding |
| `CHUNK_OVERLAP` | `150` | Character overlap between consecutive chunks |
| `PDF_DIR` | `./data/pdfs` | Directory auto-scanned for `*.pdf` on startup |
| `VECTOR_STORE_PATH` | `./data/vectorstore.json` | Where ingested chunks are persisted (set to `""` to disable persistence) |
| `MAX_UPLOAD_SIZE_MB` | `20` | Max size for `/api/ingest` uploads |

---

## 5. Run

```bash
go run ./cmd/server
```

You should see startup logs like:

```json
{"time":"...","level":"INFO","msg":"vector store loaded","path":"./data/vectorstore.json","existing_chunks":0}
{"time":"...","level":"INFO","msg":"ingested seed PDF","file":"handbook.pdf","chunks":14}
{"time":"...","level":"INFO","msg":"server starting","port":"8080","document_chunks":14}
```

Or build a binary first:

```bash
go build -o bin/go-ai-chat ./cmd/server
./bin/go-ai-chat
```

---

## 6. API reference

### `GET /api/health`
Liveness check.
```bash
curl http://localhost:8080/api/health
```

### `POST /api/chat`  <- the main endpoint you asked for
Topic-gated RAG chat. Field name is `prompt` (kept from the original DTO).

```bash
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{"prompt": "What technology is used for the CBDC ledger?"}'
```

**On-topic response:**
```json
{"answer": "The CBDC ledger uses Hyperledger Fabric, with Kafka as the event streaming backbone..."}
```

**Off-topic response** (e.g. `"What's the best pizza topping?"` when the PDF is about a CBDC platform):
```json
{"answer": "Question is not related to topic"}
```

### `POST /api/ask`
Identical behavior to `/api/chat`, field name is `question` instead of `prompt` — kept for compatibility with the original repo's DTO naming.

```bash
curl -X POST http://localhost:8080/api/ask \
  -H "Content-Type: application/json" \
  -d '{"question": "What does Kafka do in this platform?"}'
```

### `POST /api/ingest`
Upload an additional PDF at runtime (multipart form, field name `file`).

```bash
curl -X POST http://localhost:8080/api/ingest \
  -F "file=@/path/to/another-doc.pdf"
```
```json
{"source": "another-doc.pdf", "chunks_ingested": 9}
```

### `GET /api/stats`
Quick sanity check of how much is currently in the vector store.
```bash
curl http://localhost:8080/api/stats
```
```json
{"document_chunks": 23}
```

---

## 7. Postman

1. Create a new request: **POST** `http://localhost:8080/api/chat`
2. Body -> raw -> JSON:
   ```json
   { "prompt": "your question here" }
   ```
3. For `/api/ingest`, switch Body -> `form-data`, add a key named **`file`**, set its type to **File**, and pick a `.pdf`.

---

## 8. Testing

Unit/integration tests are included and run fully offline (no Ollama needed — they use fake embedder/generator ports that satisfy the same interfaces):

```bash
go test ./...
```

This covers:
- `internal/core/service` — chunker edge cases, and a full ingest -> embed -> store -> search -> topic-gate -> generate pipeline using fake ports.
- `internal/adapter/outbound/vectorstore` — persistence round-trip (write, simulate restart, reload, search).
- `internal/adapter/outbound/pdfloader` — real PDF text extraction against a bundled sample PDF.

### Manual end-to-end check
```bash
mkdir -p data/pdfs && cp your.pdf data/pdfs/
go run ./cmd/server
# in another terminal:
curl -s http://localhost:8080/api/stats
curl -s -X POST http://localhost:8080/api/chat -H "Content-Type: application/json" \
  -d '{"prompt": "<a question clearly about your PDF>"}'
curl -s -X POST http://localhost:8080/api/chat -H "Content-Type: application/json" \
  -d '{"prompt": "<a question clearly unrelated, e.g. a recipe>"}'
```

---

## 9. Tuning the topic gate

If on-topic questions are wrongly getting `"Question is not related to topic"`:
- Lower `SIMILARITY_THRESHOLD` (try `0.35`-`0.45`).
- Make sure you re-ingest if you change `EMBED_MODEL` after PDFs are already stored — old embeddings from a different model aren't comparable to new ones. Delete `data/vectorstore.json` and restart in that case.

If off-topic questions are getting real (hallucinated) answers:
- Raise `SIMILARITY_THRESHOLD` (try `0.55`-`0.65`).
- Reduce `RAG_TOP_K` so a single loosely-related chunk can't sneak in.

---

## 10. Notes / design choices

- **Vector store**: kept as an embedded JSON-persisted store rather than adding Qdrant/Chroma/pgvector, to stay dependency-free and match your existing architecture. If you later need a real vector DB service (e.g. for horizontal scaling or millions of chunks), only `internal/adapter/outbound/vectorstore` needs to change — `port.DocumentRepository` is already the seam for that.
- **PDF parsing**: `github.com/ledongthuc/pdf` is pure Go (no cgo/poppler), which keeps the build simple and fully vendorable. It handles standard text-based PDFs well; scanned/image-only PDFs would need OCR (out of scope here).
- **Topic gating happens in Go, not just via prompt instructions**: relying solely on the LLM to say "I don't know" is unreliable (models can still hallucinate). The cosine-similarity threshold check runs before the LLM is even called, which is both faster and more deterministic.
