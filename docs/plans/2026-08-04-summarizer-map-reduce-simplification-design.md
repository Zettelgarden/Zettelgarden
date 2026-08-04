# Summarizer Map-Reduce Simplification — Design

**Created:** 2026-08-04
**Status:** Design / ready to implement. Prerequisites (facts removal,
analysis-table drop) are merged.
**Tracker:** Zettelgarden-xez (beads)
**Depends on:** Zettelgarden-qsg (closed), Zettelgarden-fdi (closed)
**Touches:** `go-backend/services/summarize.go`, `go-backend/services/llmprocessor.go`,
`go-backend/handlers/summarize.go`

## Problem Statement

After the facts-removal (`qsg`) and analysis-table-drop (`fdi`) landed, the
summarizer still runs a **three-stage LLM pipeline** to produce a single
markdown blob — the only output the frontend ever consumes
(`SummarizeJobResponse { id, status, result }`, rendered with `ReactMarkdown`).

Today's pipeline (`services/summarize.go` + `services/llmprocessor.go`):

1. **`ExtractThesesAndArguments`** chunks the text (`ChunkText`, 15k-char
   chunks) and makes **one LLM call per chunk** demanding strict JSON of
   `{ section, theses[{ thesis, arguments[{ argument, importance }] }] }`.
   It tracks section transitions across chunks, merges duplicate theses, and
   runs a **JSON-repair LLM call** whenever a chunk's JSON fails to parse.
   Returns `[]SectionAnalysis`.
2. The handler serializes that `[]SectionAnalysis` into the job payload as
   nested `map[string]interface{}`, and the job re-parses it with dozens of
   type assertions (`parsePayloadAnalyses`) — a round-trip that exists only to
   hand intermediate data to the next stage.
3. **`AnalyzeAndSummarizeText`** (inside the job) makes a **dedup/rank LLM
   call** over the theses/arguments, then a **final summarization LLM call**
   that emits the markdown.

So a typical multi-chunk document costs **N + 2 LLM round-trips** (N chunk
extractions, each with a possible JSON-repair retry, plus dedup plus final),
all to produce one markdown string. The strict-JSON schema, section-transition
state machine, JSON-repair loop, and the analyses→payload→parse round-trip are
pure machinery with no consumer: facts are gone (`qsg`) and the structured
analysis is no longer persisted or read (`fdi`).

**Two additional structural problems:**

- **The expensive part runs in the request path, the cheap part runs in the
  queue.** `ExtractThesesAndArguments` (all N chunk calls) executes
  *synchronously* inside `CreateSummarizationRoute` and inside a bare
  `go func()` in `ProcessEntitiesAndFacts`. Only `AnalyzeAndSummarizeText`
  (dedup + final) is enqueued via `runSummarizationJobViaQueue`. So the job
  queue's retry/cancel/timeout protection covers the two cheap calls, not the
  N expensive ones.
- **`ProcessEntitiesAndFacts` runs in a hand-rolled goroutine** with its own
  `recover()` and no backpressure — not the job runner.

Combined size today: **~1,738 lines** across `services/summarize.go`,
`services/llmprocessor.go`, `handlers/summarize.go`.

## Goals / Non-Goals

**Goals**

- Collapse the pipeline to a classic **map-reduce**: summarize each chunk
  (map), then summarize the chunk-summaries into the final markdown (reduce).
- Eliminate the strict-JSON schema, section-transition tracking, JSON-repair
  loop, and the `analyses`→payload→`parsePayloadAnalyses` round-trip.
- Move **all** LLM work behind the job queue so it is retried, cancelable, and
  does not block the HTTP request or live in a bare goroutine.
- Cut LLM round-trips from `N + 2` to `N + 1` (and usually fewer, since map
  prompts are simpler/shorter than the JSON-extraction prompt).
- Shrink the three files to roughly **a third** of their current size.

**Non-Goals**

- Changing the **frontend contract** — `SummarizeJobResponse` stays
  `{ id, status, result }`; the result is still markdown.
- Changing the **job-queue / `JobTypeSummarization` mechanism** — we reuse the
  existing `LLMJobProcessor` + `JobRunner`; only the job's *payload* and
  *body* change.
- Touching **entities** — `ExtractSaveCardEntities` / `FindEntities` is a
  separate, in-use path and stays as-is.
- Re-introducing any structured/persisted analysis. The `summary_*` tables are
  already gone (`fdi`); they stay gone.
- Changing the rate-limit, cancel, or list/detail routes.

## Proposed Design

### New pipeline (map-reduce)

Replace `ExtractThesesAndArguments` + `AnalyzeAndSummarizeText` with two
functions:

```
SummarizeChunks(c, input) (chunkSummaries []string, usage, err)   // MAP
SummarizeReduce(c, chunkSummaries, usage) (markdown, usage, err)  // REDUCE
```

- **Map** (`SummarizeChunks`): reuse `ChunkText(input)`. For each chunk, one
  LLM call that returns a **plain-text** interim summary (no JSON). Chunks are
  independent, so this is trivially the map step. Keep the existing
  `executeLLMRequestWithRetry` wrapper (transient-failure backoff) and the
  `MaxChunkFailureRate` (50%) threshold — both still apply and are good.
- **Reduce** (`SummarizeReduce`): one LLM call that takes the chunk-summaries
  and emits the final two-section markdown (Executive Summary + Reference
  Summary). Reuse the existing final-summarization prompt verbatim — it already
  produces the desired output format.

Total round-trips: **`N + 1`** (was `N + 2`, plus the JSON-repair calls go
away entirely).

### Prompt design

- **Map prompt** (per chunk): a concise "summarize this section of a longer
  document, preserving key theses and supporting arguments, in ~150–250 words"
  instruction. Plain text output — **no JSON**, so no parsing, no repair.
  Include a one-line note that it is chunk `i of N` of a larger work so the
  model doesn't try to write an intro/conclusion per chunk.
- **Reduce prompt**: the **existing** final-summarization system prompt from
  `AnalyzeAndSummarizeText` (Executive Summary + Reference Summary, boardroom
  vs briefing tone). Its `<analysis>` block is fed the concatenated
  chunk-summaries instead of the dedup JSON.

The dedup/rank intermediate call is **dropped** — deduplication becomes the
reduce step's job, which a single good prompt handles fine for summary output.

### Job-queue placement (the structural fix)

Both entry points (`CreateSummarizationRoute` for the manual Summarizer page,
`ProcessEntitiesAndFacts` for card create/update) should do **only**: insert
the `summarizations` row (status `pending`), then enqueue one job carrying
`{ summarization_id, card_pk, input_text }`. **No LLM calls in the handler.**

`processSummarizationJob` then does the whole map-reduce:

```
input_text  (from payload or the summarizations row)
  -> SummarizeChunks   (map)
  -> SummarizeReduce   (reduce)
  -> updateSummarizationResult(status=complete, result=markdown, usage)
```

This means:

- `runSummarizationJobViaQueue` shrinks to a tiny payload — **no more
  `analyses` serialization**. The payload is just the input text + ids.
- `parsePayloadAnalyses` is **deleted**.
- `ProcessEntitiesAndFacts`'s bare `go func()` is replaced by a job enqueue;
  the `defer recover()` and `defer LinkCardToEntityIfPossible` move into the
  job (or `LinkCardToEntityIfPossible` is enqueued separately / called after).
  **Coordinate entity linking carefully** — see Risks.
- The manual `CreateSummarizationRoute` stops blocking on `ExtractThesesAndArguments`
  and returns `pending` immediately (it already returns the DB status, so the
  contract is unchanged).

### Data flow after the change

```
handler:  insert summarizations(pending) -> enqueue JobTypeSummarization
job:      load input_text -> SummarizeChunks -> SummarizeReduce
          -> UPDATE summarizations SET status='complete', result=<markdown>, usage...
```

No `SectionAnalysis`, no `analyses` payload key, no `parsePayloadAnalyses`.

## What gets deleted

In `services/summarize.go`:

- The big `ExtractThesesAndArguments` body is replaced by the much smaller
  `SummarizeChunks` (chunk loop stays; JSON parse / section-merge / repair
  blocks go).
- `AnalyzeAndSummarizeText` becomes `SummarizeReduce` (drop the dedup call and
  its prompt; keep the final-summary prompt).
- `flattenArguments`, `countTheses`, `buildContextIntro`, `buildUserContent`,
  `cleanContent` — review each; most become unused once the JSON-extraction
  path is gone.
- `models.SectionAnalysis` / `ThesisEntry` / `Argument` (in `models/summarization.go`)
  — **confirmed summarizer-private** (verified 2026-08-04: only referenced by
  `runSummarizationJobViaQueue`'s signature, the pipeline, and their own
  definitions). Once the `analyses` payload is gone they are fully unused —
  delete them.

In `services/llmprocessor.go`:

- `parsePayloadAnalyses` — deleted.
- `processSummarizationJob` simplifies to the map-reduce body (the "legacy
  path" branch that re-extracts from `input_text` becomes the *only* path).

In `handlers/summarize.go`:

- `CreateSummarizationRoute` loses its inline `ExtractThesesAndArguments` call.
- `ProcessEntitiesAndFacts` loses its goroutine + inline extraction; becomes a
  job enqueue (+ entity handling).
- `runSummarizationJobViaQueue` payload shrinks to `{ summarization_id,
  card_pk, input_text }`.

## Validation Plan

This change alters the actual summary output, so it needs a quality gate, not
just unit tests.

1. **Build a fixture set.** Pick ~10 representative inputs of varying length
   (1-chunk note, a 3-chunk article, a long transcript, an epub chapter) from
   real/anonymized content. Capture the **current** markdown output for each
   (pre-change) as the baseline.
2. **A/B compare.** After the rewrite, generate summaries for the same inputs
   and diff against baseline. Acceptance: the reduce prompt is the same, so
   structure (Executive/Reference sections) is preserved; content should be
   equivalent-or-better. Manually eyeball the 10.
3. **Unit tests.** Replace the removed tests with:
   - `SummarizeChunks`: given a stubbed `LLMClient`, returns one summary per
     chunk, accumulates usage, respects the 50% failure threshold.
   - `SummarizeReduce`: concatenates chunk-summaries and returns markdown.
   - `processSummarizationJob`: end-to-end with a stubbed client updates the
     row to `complete` with non-empty `result`.
   (The existing `removeReferences` / `prepareTextForAnalysis` tests stay —
   those helpers are unchanged.)
4. **Metrics.** Keep the `[METRICS]` log lines; after the change, confirm
   `duration`, `total_tokens`, and round-trip count drop on a multi-chunk
   input.

## Coordination / Sequencing

- **Wait for the Postgres-retirement work to land first.** As of 2026-08-04 a
  concurrent workstream (epic `Zettelgarden-c7j` Phase 7b) has a large
  uncommitted changeset in the tree: `server/database.go`,
  `tests/conftest.go`, `schema/`, `cmd/migrate-pg-to-sqlite` removal, lib/pq
  removal. It touches files adjacent to the summarizer. Starting the behavioral
  rewrite on top of in-flight changes invites messy merges. Land 7b, then
  branch `xez` off a clean `master`.
- After 7b lands, also add the **orphan-table drop** that `fdi` deferred: an
  idempotent `DROP TABLE IF EXISTS summary_{arguments,theses,sections}` in
  `ensureSQLiteSchemaUpgrades` (`server/database.go`) so existing DBs converge
  on the consolidated schema. (`fdi` removed them from fresh-DB schema only.)
- Consider renaming `ProcessEntitiesAndFacts` → `ProcessCardSummary` (it no
  longer processes facts; the name is now misleading). Cosmetic, low priority.

## Risks / Rollback

- **Entity linking timing.** `ProcessEntitiesAndFacts` currently guarantees
  `LinkCardToEntityIfPossible` runs once (via `defer`) after extraction. Once
  extraction moves to the job, decide: enqueue entity extraction/linking as a
  separate job, or run `ExtractSaveCardEntities` synchronously in the handler
  (it's a separate `FindEntities` call, independent of summarization). Simplest:
  keep `ExtractSaveCardEntities` + `LinkCardToEntityIfPossible` in the handler
  path (they don't depend on the summary), and let only the summary go to the
  queue. **Confirm this is acceptable before implementing.**
- **Summary quality regression.** The dedup/rank step is being removed. If A/B
  testing shows worse dedup on long inputs, the fallback is to keep a single
  reduce pass but feed it the raw chunk-summaries and instruct dedup inline
  (cheap; no extra call).
- **`SectionAnalysis` model ownership.** Confirmed summarizer-private (see
  above) — safe to delete with the rewrite.
- **Rollback.** The change is isolated to the summarizer call path. If it
  regresses, revert the single commit; the `summarizations` table/contract is
  unchanged so no data migration is involved.

## Acceptance Criteria

- [ ] `go test ./...` green; removed tests replaced by map/reduce tests.
- [ ] A/B comparison on the 10-fixture set shows equivalent-or-better markdown.
- [ ] No LLM calls in `CreateSummarizationRoute` or `ProcessEntitiesAndFacts`
      (both just insert + enqueue).
- [ ] No `analyses` key in the job payload; `parsePayloadAnalyses` deleted.
- [ ] `services/summarize.go` + `handlers/summarize.go` + the summarizer parts
      of `llmprocessor.go` roughly a third of their pre-`qsg` size.
- [ ] `[METRICS]` logs show fewer round-trips and lower token totals on a
      multi-chunk input.
