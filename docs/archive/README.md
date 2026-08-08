# Docs Archive

One-off reports, migration-era summaries, and marketing assets archived during
the documentation audit (2026-08-08, [Zettelgarden-0ui]).

These documents are **historical**: they describe an earlier state of the app
(SaaS-era positioning, PostgreSQL database, Backblaze B2 storage, one-off
feature/test reports) and should not be read as a description of the current
app. They are kept here for the record; git history preserves the full content.

| Subdir | Contents |
| ------ | -------- |
| `marketing/` | SaaS-era marketing docs (marketing copy, competitive analysis, product overview, feature specifications). These belong to the extracted landing-page/blog projects (epics 6er.13/6er.14), not this repo. |
| `reports/` | One-off feature/test/migration reports and summaries (entity fix, RSS starring test report, React Query migration docs, API error handling migration, SQLite feasibility analysis, Phase-3 domain-package migration records, MobileTopBar implementation summary, EmailDetailPage testing checklists). |
| `root/` | Stale root-level artifacts: `TODO.md` (abandoned AI-agent multi-user task list referencing a handoff doc that never existed) and the one-off entity-fix summary + memory-job SQL fix script. |

## Archived contents at a glance

- **TODO.md** — tracked the abandoned "AI Agent Multi-User Support" effort
  (tasks 1–7/16). The referenced handoff doc
  (`docs/agent-implementation-handoff.md`) never existed. Agent tables exist in
  the schema but no agent handlers/routes were ever added; the feature was
  never completed. Revisit via beads if the feature is ever revived.
- **2026-07-02-sqlite-feasibility-analysis.md** — explored running
  PostgreSQL + SQLite from one backend. The decision that followed (epic
  Zettelgarden-c7j) was SQLite-only; PostgreSQL was retired 2026-07-28.
- **API_ERROR_HANDLING_* / PACKAGE_JSON_CHANGES.md / REACT_QUERY_*** —
  one-off migration summaries for frontend refactors that have since landed.
  `zettelkasten-front/REACT_QUERY_QUICK_START.md` was kept in place as a living
  dev reference.
- **2026-02-27-* (task-8 report, quick-testing checklist)** — EmailDetailPage
  verification artifacts; that page no longer exists.

Newly archived files carry an `ARCHIVED` header. In-place plans under
`docs/plans/` that describe pre-SQLite/pre-local-storage infrastructure carry a
`HISTORICAL`/`SUPERSEDED`/`EXECUTED` header instead of being moved.
