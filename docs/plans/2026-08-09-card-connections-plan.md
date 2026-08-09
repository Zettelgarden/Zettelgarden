# Card Connections: Feature Brainstorm & Plan

**Status:** Draft — brainstorming + phased roadmap
**Date:** 2026-08-09
**Related:** `docs/plans/2026-02-12-related-cards-design.md` (initial related-cards feature)

## 1. Goal

Zettelgarden's core value is building a connected web of notes (a zettelkasten). Today we compute relatedness but surface very little of it, and we provide almost no tools for *actively* growing the network. This plan audits what exists, identifies gaps, and lays out a phased roadmap — from cheap explainability wins to a full interactive graph.

## 2. Current State Audit

### What exists today

| Capability | Where | Notes |
|---|---|---|
| Wiki-link backlinks `[[card_id]]` | `go-backend/services/backlink/backlink.go`, `backlinks` table | Categorized into bidirectional / incoming / outgoing in the rail |
| References section | `ViewPageSidePanels.tsx` → "Linked references" | Shows all 3 categories + `BacklinkInput` to add links |
| Parent/child hierarchy | `cards.parent_id`, ChildrenCards | Tree navigation in header |
| Related cards endpoint | `GET /api/cards/{id}/related` (`handlers/cards.go`) | Scores: shared entity +3 each, shared tag +1 each, semantic (Typesense) scaled to 0–10; excludes self/parent/children/siblings/references; top 10 |
| Related cards UI | `RelatedCards.tsx` (desktop rail + mobile accordion) | Shows title + score only |
| Entities & facts | `entities`, `entity_card_junction`, `facts` | Auto-extracted per card; drive entity-based relatedness |
| Semantic search | Typesense embeddings (`server/similarity.go`) | Graceful fallback when Typesense absent |

### Gaps found in the audit

1. **`reasons` are computed but never shown.** The backend returns `Reasons []string` per related card (e.g. `["entities","tags","similarity"]`) and the TS model has `reasons: string[]` — but `RelatedCards.tsx` renders neither reasons nor matched entity/tag names. Relatedness is a black box → users don't trust or act on it.
2. **Weights & limits are hardcoded.** The 2026-02-12 design promised `RELATED_ENTITY_WEIGHT`, `RELATED_TAG_WEIGHT`, `RELATED_MAX_RESULTS` env vars; they were never implemented (3 / 1 / 10 are hardcoded in the handler).
3. **Related cards are fetched once and never refreshed.** `ViewPageContainer` fetches on card load and caches; editing the body does not recompute.
4. **No graph visualization.** Obsidian's signature feature; Zettelgarden has zero network visualization.
5. **No unlinked mentions.** Cards that *mention* another card's `card_id`/title in plain text (without `[[...]]`) are invisible — the highest-value "connect these" signal in Roam/Logseq.
6. **No orphan / low-connectivity report.** No way to find cards with no links, no children, no references.
7. **No second-degree ("people who linked to X also linked to…") suggestions.**
8. **Entity page is management-only.** `EntityPage.tsx` lists/merges/deletes entities; it does not show the web of cards connected through an entity (an "entity hub"). (Partial credit: `EntityTable` already shows `card_count` + one linked card, and `EditEntityDialog` has a `BacklinkInput` — Phase 4.1 extends this surface rather than building from scratch.)

### Schema note (verified by review)

The `backlinks` table (`go-backend/schema/sqlite/schema.sqlite.sql`) has **no `user_id` column and no primary key** — rows are `source_id_int → target_id_int`. Any new query against it (orphans, unlinked mentions, path BFS) must JOIN `cards` for ownership filtering (as `GetBacklinks` already does at `services/cards.go:444`) and dedupe manually.
9. **Related cards are read-only on ViewPage.** Only navigation + "+Ref". No "why", no grouping, no direct "add link" affordance in context.
10. **No connection surface in the editor.** While writing, no live suggestions of cards you probably want to link.
11. **No connection tooling in MCP / zg CLI / desktop.** `zettelgarden-mcp` and `zg` can't ask "what's related to X?" or "which cards should I connect?"

## 3. Brainstorm — Feature Catalog

Organized by theme; each item is a candidate. Items marked **[quick win]** are small, self-contained, and high value.

### A. Explainability & quality (make relatedness trustworthy)

- **A1. Show why.** Render matched entities/tags + similarity as chips/labels on each related card ("Shares 3 entities: Python, LLM", "tag: research", "semantic").
- **A2. Configurable scoring.** Env vars for entity weight, tag weight, semantic weight, max results.
- **A3. Live refresh.** Recompute related cards when the card body is saved.
- **A4. Group by reason.** Related Cards section groups results into "Shared entities", "Shared tags", "Semantically similar".
- **A5. Similarity without Typesense.** Fall back to a simple TF/overlap score (shared rare tokens between title+body) when Typesense is off — keeps self-hosted installs useful.

### B. Active discovery (help users *find* connections to make)

- **B1. Unlinked mentions.** For a card, find other cards whose body mentions its `card_id` or title without `[[...]]`. One-click "add link" (inserts `[[card_id]]` into the mentioning card). Endpoint `GET /api/cards/{id}/unlinked-mentions`.
- **B2. Orphan report.** List cards with zero incoming+outgoing references, no children, low/absent relatedness — the "content gardening" surface. Could live in StatsPage and/or a rail tab.
- **B3. Second-degree suggestions.** "Cards referenced by cards that reference this card" — classic collaborative signal. `GET /api/cards/{id}/suggestions`.
- **B4. Same-source grouping.** Cards from the same RSS source_article / link domain, surfaced under related.
- **B5. Editor-time link suggestions.** In `CardBodyTextArea`/editor, live "cards you might want to link" strip; `[[` autocomplete already exists (`BacklinkInput`), extend it with related candidates.
- **B6. Global "connect these" queue.** A page listing the strongest unlinked pairs across the whole vault (top-N by shared entities/tags minus existing links).

### C. Visualization & navigation (make the network *visible*)

- **C1. Interactive knowledge graph.** New `/app/graph` page: nodes = cards (optionally entities/tags), edges = link type (reference / parent-child / shared-entity). Pan/zoom, drag, click-through, filter by tag/type, search-to-focus. Library: d3-force or `@xyflow/react` (React Flow) — both fit the current React 18 stack.
- **C2. Local ego-network in the rail.** 1–2 hop neighborhood of the viewing card rendered as a mini graph or radial list, replacing/augmenting the flat Related Cards list.
- **C3. Path finding.** "How is card A connected to card B?" — shortest path in the link graph (BFS; vaults are small, this is cheap in Go or JS).
- **C4. Breadcrumb trail while exploring.** When clicking through related cards, keep a visible "why I came here" chain.

### D. Network-level intelligence

- **D1. Entity hubs.** Entity detail view (reuse `DialogStateContext` selected-entity flow) listing all cards connected through that entity + the entity's own network stats.
- **D2. Clustering.** Tag/entity co-occurrence clusters surfaced in graph view (color nodes by cluster).
- **D3. Network health stats.** In StatsPage: avg links per card, orphan count, top-10 connector cards, link growth over time.

### E. Outward surfaces (AI + CLI + desktop)

- **E1. MCP tools.** `related_cards`, `unlinked_mentions`, `suggest_links` tools in `zettelgarden-mcp/src/tools.py` so the AI assistant can actively build connections.
- **E2. zg CLI.** `zg related <card>`, `zg orphans`, `zg suggest` commands.
- **E3. Desktop app.** Graph view + connection panels in the Tauri client (rides on the sync engine's local SQLite).

## 4. Prioritized Plan

Sequencing logic: Phase 1 fixes the existing feature's trust problem (reasons, config, refresh) with tiny diffs. Phase 2 adds the highest-value *active* discovery features (unlinked mentions, orphans, editor suggestions) — these are the features that actually grow connections. Phase 3 makes the network visible and explorable (graph). Phase 4 layers on network intelligence and outward surfaces. Each phase is independently shippable.

### Phase 1 — Explainability & scoring control *(small, ~1–2 days)*

- **1.1 [A1]** `RelatedCards.tsx`: render `reasons` as small chips/labels. Backend: enrich `Reasons` with matched entity/tag *names* (currently only categories). Note: `GetCardsBySharedEntities`/`GetCardsBySharedTags` (`services/cards.go:1318,1378`) return `map[int]int` **scores only** — collecting names requires a new query against `entity_card_junction`/`card_tags` per candidate (batched `WHERE card_pk IN (...)`). The payload shape changes (today reasons are deduped to categories at `handlers/cards.go:1543`), so backend + `Card.ts` `RelatedCard` model + `RelatedCards.tsx` must change together.
  - Files: `go-backend/handlers/cards.go`, `go-backend/models/card.go` (comment), `zettelkasten-front/src/models/Card.ts`, `zettelkasten-front/src/components/cards/RelatedCards.tsx`, `ViewMobileLayout.tsx` (inherits automatically).
- **1.2 [A2]** Read `RELATED_ENTITY_WEIGHT`, `RELATED_TAG_WEIGHT`, `RELATED_SEMANTIC_WEIGHT`, `RELATED_MAX_RESULTS` from env in `GetRelatedCardsRoute`; the semantic `sc.Score * 10.0` scale (`handlers/cards.go:1469`) is currently inline and must use the new var. Document all four in `.env.example`.
- **1.3 [A3]** In `ViewPageContainer`, refetch related cards when `viewingCard` body/title changes (after save). Invalidate the `relatedCards` state on save like other data.
- **1.4 [A4]** Group related results by top reason in the rail (three small sublists) — only when >1 reason present, to avoid clutter.

### Phase 2 — Active discovery *(medium, ~3–5 days)*

- **2.1 [B1] Unlinked mentions.**
  - Backend: `GET /api/cards/{id}/unlinked-mentions` → scan other cards' bodies for the source `card_id` (v1: card_id only — title-token matching is too noisy: "Python" substrings hit URLs/code/markdown; add word-boundary + min-length + stopword rules only if card_id alone feels thin). Return `{ card, mention_count, context_snippet }[]`. SQL `LIKE` scan + regex filter in Go; reuse `backlink.ExtractBacklinks` to exclude already-linked (note: it covers both `[[card_id]]` and legacy `[card_id]` syntax at `backlink.go:46-63`). Ownership filtering must JOIN `cards` (backlinks table has no `user_id`).
  - Frontend: new collapsible "Unlinked mentions" in the rail's Links tab + mobile accordion; each row has an "Add link" button that inserts `[[card_id]]` into the mentioning card. Reuse the existing `addBacklinkToCard` helper (`utils/cardActions.ts:168`) — it inserts the link and saves, triggering `UpdateBacklinks` regeneration — rather than raw `saveExistingCard`.
- **2.2 [B2] Orphan report.**
  - Backend: `GET /api/cards/orphans` → cards with 0 refs in/out (JOIN `cards` for ownership; dedupe manually — no PK), 0 children, low shared-entity/tag overlap. **Do NOT run the full related-score (it triggers a Typesense semantic call per card at `handlers/cards.go:1438`)**; use a cheap entity/tag-overlap-only proxy or a precomputed offline score.
  - Frontend: section on StatsPage or a dedicated "Network" tab; list with one-click "find related" deep-link.
- **2.3 [B5] Editor link suggestions.**
  - In `CardEditor`/`CardBodyTextArea`, compute related candidates on debounce while typing; show a slim "Link these" strip under the editor (reuse `CardItem` + click inserts `[[card_id|title]]`).
  - Reuses `getRelatedCards`; no new backend work. **Scope note:** `getRelatedCards` needs a card ID, and `EditPage` also serves `/app/card/new` (`AppRoutes.tsx:84`) with no ID yet — gate the strip on an existing card.
- **2.4 [B3] Second-degree suggestions.** Extend the related endpoint (or new `GET /api/cards/{id}/suggestions`) with one hop: refs of refs, scored by overlap; exclude already-linked.

### Phase 3 — Network visualization *(large, ~1–2 weeks)*

- **3.1 [C1] Graph page.** New route `/app/graph` + `GraphPage.tsx`.
  - Backend: `GET /api/graph?depth=2&types=...` returning nodes (cards/entities/tags) + edges (reference, parent, shared-entity, shared-tag) for the user's vault (paginated/streamed by tag filter). Every new endpoint must go through `addProtectedRoute` (`routes/helpers.go:20` — auth sets `current_user`, so filter all queries by it). Put the route in a new `routes/graph.go` rather than crowding `routes/cards.go`.
  - Frontend: React Flow (`@xyflow/react`) canvas — **not currently a dependency** (needs a `npm install`; React 18 compat is fine); card nodes show title + card_id; click → navigate to ViewPage; filter by tag/type; search box focuses a node; entity/tag nodes expandable.
  - Sidebar entry + mobile fallback (read-only list mode on small screens).
- **3.2 [C2] Ego-network in the rail.** A "Network" mini-graph (or radial 2-hop list) replacing the flat Related Cards list when open. Reuses `getRelatedCards` + references data already in the container — mostly a frontend change.
- **3.3 [C3] Path finding.** `GET /api/cards/{from}/path/{to}` — BFS over an **undirected adjacency built from both `backlinks` columns (`source_id_int`, `target_id_int`) plus `cards.parent_id`** (backlink rows are single-direction; don't forget to reverse). UI in graph page (click two nodes → highlight path) and/or a small "Connect A ↔ B" dialog on ViewPage.

### Phase 4 — Network intelligence & outward surfaces *(medium-large)*

- **4.1 [D1] Entity hubs.** Extend the existing entity surface (not from scratch): `EntityTable` already shows `card_count` + one linked card, and `EditEntityDialog` has a `BacklinkInput` (`components/entities/EditEntityDialog.tsx`). Upgrade the entity detail dialog to list all cards via `entity_card_junction` with shared-entity count; link out to ViewPage.
- **4.2 [D3] Network stats.** StatsPage section: link counts, orphan list summary, top connectors.
- **4.3 [E1] MCP tools.** Add `related_cards`, `unlinked_mentions`, `suggest_links` to `zettelgarden-mcp/src/tools.py` hitting the new endpoints; wire into the AI agent so it can propose connections in chat.
- **4.4 [E2] zg CLI.** `zg related`, `zg orphans`, `zg suggest` in `zg/` (rides on the same REST API).
- **4.5 [D2] Clustering** (optional): color graph nodes by tag co-occurrence clusters.

## 5. Non-goals / deferred

- Typed/labeled links ("supports", "contradicts") — heavy schema change; wiki-link + entity network covers most zettelkasten needs.
- Block-level linking (Roam-style) — conflicts with the card-as-unit model.
- Collaborative/shared graphs — single-user, self-hosted focus.
- Auto-linking without user action — suggestions only; we never rewrite bodies silently.

## 6. Testing notes

- Backend: table-driven tests for scoring merge (entity+tag+semantic), exclusion logic, unlinked-mention detection (link vs plain mention), orphan query, path BFS.
- Frontend: Vitest for `RelatedCards` reason rendering + editor suggestion strip; graph page smoke test (render, filter, navigate).
- Integration: new endpoints mirrored in `go-backend/handlers/cards_test.go` fixtures under `go-backend/tests/`.

## 7. Open questions

1. Graph library choice: `@xyflow/react` (React Flow, maintained, MIT) vs `d3-force` custom — recommend React Flow for interaction ergonomics; confirm bundle-size appetite (needs adding to `package.json`).
2. Should unlinked-mention scanning run on demand only, or be indexed into the `backlinks` table as a "possible" link type? (Recommend on-demand for Phase 2; index later if slow.)
3. Where should the orphan report live: StatsPage vs. dedicated Network page (bundled with the graph in Phase 3)?

## 8. Review status & kickoff

Reviewed by the project `reviewer` agent (2026-08-09). Audit claims verified accurate against the codebase; the warnings above (reasons-enrichment SQL, `backlinks` schema ownership, orphan-score perf, editor-suggestion scope for new cards, env-var consistency) are folded into the phases. Per AGENTS.md, each phase should be filed as **bd issues** at kickoff rather than tracked in this doc.
