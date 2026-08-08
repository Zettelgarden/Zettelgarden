> **STATUS: HISTORICAL — pre-SQLite era.** This plan predates the PostgreSQL→SQLite cutover (2026-07-28, epic Zettelgarden-c7j) and the move to local on-disk file storage (epic Zettelgarden-yar). Zettelgarden now runs SQLite-only with local storage; this document is kept for design history.

# Vintage OmniFocus 1 (OS X Tiger Aqua) — Design Document

**Date:** 2026-07-02
**Status:** Proposal
**Author:** (planning session)

## Overview

Build a **web-based task manager with the look and feel of OmniFocus 1 running on Mac OS X 10.4 "Tiger" (Aqua UI, ca. 2005–2008)**, using Zettelgarden as the backend.

The premise is that Zettelgarden's task subsystem is already a near-complete OmniFocus data model in disguise. This means we can build a faithful retro skin **on top of the existing API with little to no backend work** for an MVP, then layer in the missing pieces (Perspectives UI, review mode, etc.).

The deliverable is a separate front-end app (or an isolated route in the existing app) that renders the classic three-pane OmniFocus 1 layout with brushed-metal windows, pinstriped lists, glossy gel buttons, blue translucent source list, traffic-light window controls, and Lucida Grande typography.

---

## 1. The big insight: Zettelgarden ≈ OmniFocus data model

Almost every OmniFocus 1 primitive already exists in the ZG schema/API:

| OmniFocus 1 concept         | Zettelgarden equivalent                                              | Status |
|-----------------------------|----------------------------------------------------------------------|--------|
| **Action**                  | `Task` (`models/tasks.go`)                                           | ✅ exists |
| **Action group** (nested)   | `parent_task_id` + `subtasks`                                        | ✅ exists |
| **Project**                 | Top-level task with children, or a parent **Card**                   | ✅ exists |
| **Inbox**                   | Tasks with no project/parent and no context                          | ✅ derivable |
| **Contexts**                | `tags` (`Task.tags`, `models/tags.go`)                               | ✅ exists |
| **Status** (Active / On Hold / Completed / Dropped) | `task_statuses` (user-defined, colored)              | ✅ exists |
| **Flagged**                 | `priority` (e.g. a "flagged"/"high" value)                           | ✅ exists |
| **Due date**                | `due_date`                                                           | ✅ exists |
| **Defer (start) date**      | `scheduled_date`                                                     | ✅ exists |
| **Repeating actions**       | `recurrence expansion` (migrations `0110`/`0130`)                    | ✅ exists |
| **Dependencies**            | `blocked_by` / `blocks` + `/dependencies` routes                     | ✅ exists |
| **Perspectives** ⭐          | `task_saved_searches` (filter + sort + **view mode**, synced!)        | ✅ exists |
| **Note / attachment**       | The linked `Card` (zettel) behind each task                          | ✅ exists |
| **Estimates**               | —                                                                    | ⚠️ no field (could stash in description / a tag) |
| **Review mode**             | —                                                                    | ❌ new UX only |
| **Forecast (calendar)**     | External calendar tables exist (`external_events`)                   | 🟡 partial |

The headline: **`task_saved_searches` is literally OmniFocus Perspectives** — it persists `filter_string`, `sort_field`, `sort_direction`, and `view_mode` per user and syncs across devices. This is the single most important alignment.

**Conclusion:** No backend changes are required for a credible v1. We build a new view over the existing `/api/tasks`, `/api/task-saved-searches`, `/api/task-statuses`, and `/api/tags` endpoints.

---

## 2. Architecture decision

**Recommended: a separate Vite + React + TypeScript app**, `omnifocus-classic/` (or `vintage/`) at the repo root.

Why separate, not a route inside `zettelkasten-front/`:
- The existing app has a mature Tailwind design system; a Tiger skin would fight it. Isolation keeps both clean.
- A dedicated app can ship its own global Aqua stylesheet, fonts, and window chrome without leaking.
- Smaller bundle, faster cold start, easy to deploy as a standalone "app" (it *feels* like an old Mac app, which is the point).

What we **reuse** from `zettelkasten-front/` by importing as source (not duplicating):
- `src/api/*` — the typed API client (`tasks.ts`, `taskSavedSearches.ts`, `taskStatuses.ts`, `tags.ts`, `auth.ts`, `client.ts`, `queryClient.ts`).
- `src/models/*` — `Task.ts`, `TaskSavedSearch.ts`, `TaskStatus.ts`, `Tags.ts`, `Auth.ts`.
- `src/utils/taskDataProcessing.ts`, `src/utils/tasks.ts`, `src/utils/dates.ts` — the filter DSL evaluation, date helpers.
- Auth flow / JWT handling.

A thin shared package boundary: either path alias (`@zg/...`) or a small `shared/` workspace package. Path aliases are the lowest-friction start.

**Alternative (not recommended for v1):** an `/of1` route in the existing app, scoped via a separate CSS reset so global Aqua styles only apply under that route. Faster to start but risks style collisions and muddies the main app.

---

## 3. The Tiger Aqua design system

The aesthetic target is Mac OS X 10.4 "Tiger" (2005) as seen in OmniFocus 1 (2008). Core ingredients and how to reproduce them in CSS:

### 3.1 Window chrome
- Rounded-corner window with soft drop shadow (`border-radius: 10px`, layered `box-shadow`).
- **Unified title bar**: a vertical gradient (light → slightly darker), 1px hairline border, traffic-light buttons top-left (red/yellow/green circles with an inner highlight).
- **Brushed metal** background option: a `repeating-linear-gradient` of very subtle horizontal grey lines over a noise texture (SVG `feTurbulence`) — the classic Finder/iTunes metal.
- Optional **drawer** panels that slide in from the side.

### 3.2 Lists & pinstripes
- Alternating row colors via `:nth-child(even)` — the iconic pale-blue pinstripe tables (`#e8eef5` / `#ffffff`).
- Row hover and selection: the translucent blue selection (`rgba(40,90,180,.25)` with an inset highlight) of Tiger.
- **Disclosure triangles** for action groups / projects (rotated CSS triangles via `border` tricks).
- Circular blue checkboxes (the Aqua radio/checkbox look) instead of square ones.

### 3.3 Source list (left sidebar)
- Translucent blue-grey gradient background with white text, the Finder/iTunes "Source List" look.
- Grouped sections: **LIBRARY** (Inbox, Projects, Contexts, Flagged, Review), **PERSPECTIVES** (saved searches), **CONTEXTS** (tags).
- Selected item gets the rounded translucent-white highlight.

### 3.4 Gel / candy buttons & controls
- Glossy "Aqua" push buttons: multi-stop vertical gradient (white cap → blue → darker blue) with a 1px inner highlight line at the top and a soft bottom glow. Active = depressed (translateY + inset shadow).
- Pill-shaped segmented controls for the perspective view-mode switcher (List / Grouped / …).
- Blue gel scrollbars (custom `::-webkit-scrollbar` with gradients) to complete the illusion.

### 3.5 Typography & iconography
- Font stack led by **Lucida Grande**, with fallbacks to Tahoma/Verdana for non-Mac (bundle the webfont to guarantee fidelity).
- Toolbar: 32px glossy icon set — can source from open Aqua icon packs (CC-licensed) or commission a small set (Inbox, Projects, Contexts, Flagged, Review, Forecast, +).
- Toolbar style: large icons with labels under them, the classic OF1 toolbar.

### 3.6 Implementation notes
- Deliver the theme as **plain CSS + a handful of SVG/gradient assets** — no heavy framework. A single `aqua.css` plus component-scoped `.module.css` files.
- Provide a `Window` primitive (title bar + traffic lights + draggable title), a `SourceList`, a `PinstripeList`, a `GelButton`, a `DisclosureTriangle`, a `SegmentedControl`, a `AquaCheckbox`.
- Keep it responsive enough for desktop; full Tiger window chrome assumes a wide screen. A "modern fallback" for narrow viewports is out of scope for v1.

---

## 4. Component breakdown (OF1 three-pane layout)

```
┌─────────────────────────────────────────────────────────────┐
│ ●●●   OmniFocus — [Perspective Name]            [toolbar]    │  ← Window/TitleBar
├──────────┬───────────────────────────────────┬──────────────┤
│ LIBRARY  │  ▾ Project A                       │  Title       │
│  Inbox   │    ☐ Action one      ⚑  ⏰ due     │  [notes]     │
│  Proj…   │    ▾ Action group                  │  Tags        │
│  Contexts│      ☐ Sub-action                  │  Status      │
│ PERSPEC… │  ▾ Project B                       │  Defer/Due   │
│  Today   │    ☐ ...                           │  Depends on  │
│  ...     │                                    │              │
└──────────┴───────────────────────────────────┴──────────────┘
   Source        Outline (main)                    Inspector
```

Core components to build:
1. **`AppShell`** — renders the `Window`, wires auth + data fetching (React Query, reusing `queryClient`).
2. **`Sidebar` / `SourceList`** — Library groups + dynamic Perspectives list (from `task_saved_searches`) + Contexts (from `tags`). Selecting an item sets the active perspective/query.
3. **`Outline`** — the pinstriped task tree. Renders action groups recursively (`subtasks`), checkboxes, due/defer chips, flag, context tag, status color dot. Inline editing of title; disclosure triangles.
4. **`Inspector`** — the right-hand detail pane for the selected task: notes (renders the linked **Card**'s markdown), status picker, defer/due, tags/contexts, dependencies (`blocked_by`/`blocks`), priority/flag, recurrence.
5. **`Toolbar`** — glossy icon toolbar: Perspective switcher, Add Action, Add Project, Collapse All, View Mode segmented control, Inspect toggle.
6. **`PerspectiveBar`** — shows current perspective name + its filter/sort/view mode; "Save as Perspective" → `task_saved_searches` create.
7. **`QuickEntry`** — the classic OF1 quick-entry floater (⌥␣): title + notes + project + context + dates. Calls `POST /api/tasks`.
8. **`FlaggedBadge` / `InboxBadge`** — counts in the source list.

### Mapping the perspective filter to the outline
The existing saved search stores a `filter_string` DSL evaluated client-side (see `utils/taskDataProcessing.ts` and `TaskDesktopLayout.tsx`). We reuse that evaluator verbatim so OF1 perspectives and the modern ZG task page stay **interoperable and synced**. A perspective created in the retro app shows up in the modern app and vice versa.

---

## 5. Phased roadmap

### Phase 0 — Spike (≈1–2 days)
- Stand up `vintage/` Vite app, wire auth + import shared `api/` + `models/`.
- Render a hard-coded `Window` with brushed metal, fetch `/api/tasks`, dump them in a pinstriped list.
- **Goal:** prove the Aqua CSS + API wiring in one screen.

### Phase 1 — Three-pane MVP (the core experience)
- `SourceList` with Inbox / Projects / Contexts / Flagged (derived from tasks), plus Perspectives from saved searches.
- `Outline` with nested action groups, Aqua checkboxes, due/defer/flag/context chips, inline title edit, completion.
- `Inspector` showing notes + dates + tags + status + dependencies.
- Create/complete/delete actions hitting the live API.
- **Definition of done:** you can run a full GTD day in the retro UI.

### Phase 2 — Perspectives fidelity
- Full perspective bar: name, filter editor, sort, view mode; "Save as Perspective" / "Edit Perspective".
- Reuse the shared filter DSL evaluator (interoperable with modern app).
- Perspective switching keyboard shortcuts (OF1: ⌘1–9).
- Default built-in perspectives (Inbox, Projects, Contexts, Flagged, Due Soon, Review) implemented as canned filters.

### Phase 3 — OF1 power features
- **Review mode** (new, client-only): iterate projects by last-reviewed date, mark reviewed.
- **Forecast** view: tasks grouped by `due_date`, optionally merging `external_events`.
- **Quick Entry** floater with global hotkey.
- Repeating-task awareness (show recurrence glyph; rely on backend expansion).
- Estimates: decide representation — propose a `#estimate:30m` tag convention to avoid schema changes, or a small migration if we want first-class.

### Phase 4 — Polish & delight
- Sound effects (optional): the Tiger "pop"/"bubble" check-off sound on completion.
- Animations: genie/scale window open, slide-in drawers, the bounce on new item.
- Print stylesheet that mimics the OF1 outline printout.
- Theme fidelity pass: traffic-light behaviors, sheet-style modal dialogs sliding down from the title bar.

---

## 6. Reuse inventory (do not rewrite)

From `zettelkasten-front/src/`:
- `api/client.ts`, `api/queryClient.ts`, `api/tasks.ts`, `api/taskSavedSearches.ts`, `api/taskStatuses.ts`, `api/tags.ts`, `api/auth.ts`
- `models/Task.ts`, `models/TaskSavedSearch.ts`, `models/TaskStatus.ts`, `models/Tags.ts`
- `utils/taskDataProcessing.ts` (filter/normalize), `utils/dates.ts`, `utils/tasks.ts`
- Auth/token handling from `contexts/AuthContext.tsx`

Backend (no changes for v1): `routes/tasks.go`, `routes/task_saved_searches.go`, `routes/taskstatuses.go`, `routes/tags.go`, `models/task_saved_search.go`.

---

## 7. Open questions / decisions to make

1. **Same auth or separate?** Reuse ZG JWT auth (recommended) so it's "another view of your garden." Confirm the token cookie/header approach works for a separately-hosted origin (CORS / cookie domain).
2. **Hosting.** Separate domain (`focus.zettelgarden.com`) or a path on the same host? Affects auth cookie sharing.
3. **Estimates.** Tag convention now, migration later? (No schema impact for v1.)
4. **"Projects" representation.** Top-level task-with-children, or a designated Card type? Recommend top-level task with a `#project` tag or a status for v1; revisit if a Projects perspective needs metadata.
5. **License of Aqua icon assets.** Use CC-licensed icon packs vs. an original minimal set to avoid trademark/asset issues before any public deploy.
6. **Webfont licensing** for a Lucida-Grande-equivalent (or a libre lookalike such as a tuned "SF Hello"/"Seattle"/Tahoma fallback).

---

## 8. Risks

- **Style fidelity vs. effort.** Pixel-perfect Tiger chrome is a lot of CSS; we should timebox Phase 0 to validate the look before committing.
- **Asset/legal.** Apple Aqua is a known trademark aesthetic — fine for a personal/private tool, but worth noting if this is ever public/marketed.
- **Shared-code coupling.** Importing `zettelkasten-front/src` into a second app creates an implicit dependency; a `shared/` package is the cleaner long-term move once the spike proves out.
- **Filter DSL drift.** If the two apps ever diverge on the `filter_string` grammar, perspectives break cross-app. Mitigation: import the single evaluator, never copy it.

---

## 9. Immediate next step

Approve this design, then kick off **Phase 0 spike**: create `vintage/`, wire auth + tasks API, and produce one screen that *looks* like Tiger rendering your real Inbox. That single screenshot is the decision point for everything else.
