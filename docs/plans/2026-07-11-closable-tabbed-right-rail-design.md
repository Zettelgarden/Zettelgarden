# Closable, Tabbed Right Rail for Card View — Design & Implementation Plan

**Date:** 2026-07-11
**Status:** Proposed
**Related:** Builds on the ViewPage simplification (commits `bebb0480` → `79d8f302`, Jul 2026)

## Overview

Make the card view's right-hand sidebar behave like Obsidian's: closable, tabbed, and able to host different "views." As a first concrete move, pull **Children** and **Linked references** out of the main column and into a **Links** tab in that rail — so the reading surface becomes pure card content (body, tasks, entities/files/history), while all the relationship data lives one glance to the right.

This is a handoff plan. It captures the design decisions, the existing infrastructure to reuse, and a phased rollout so the next agent can pick it up without re-deriving anything.

## Context: where things stand now

The ViewPage was just simplified across six commits. The current shape:

- **`ViewPage.tsx`** (~250 lines) — a thin shell that branches into `ViewMobileLayout` on mobile and a desktop two-column layout otherwise. `viewMode` lives in `useViewPageContainer` and is `'normal' | 'summary' | 'analysis'` (tree was deleted).
- **Main column** (`ViewCardContentSection.tsx`) — renders `CardBody`, then a **Children** section, then a **collapsible "Linked references" block** (built with the new `Collapsible` component), then Tasks, then the `ViewCardTabbedDisplay` (entities/files/history/summaries).
- **Right rail** (`ViewPageSidePanels.tsx`) — one always-visible scrolling column containing, in order: Parent + sibling nav, Source article, Linked entities, Related cards, Structured data, Tags, Details.
- **Header** (`ViewPageHeader.tsx`) — Obsidian-style title block + segmented view-mode control + quiet star/pin icon toggles + overflow menu.

The right rail currently mixes seven concerns at equal visual weight, and can't be hidden. That's the problem this plan solves.

## Existing infrastructure to reuse (do NOT reinvent)

Before building anything new, the next agent should know these already exist:

1. **`UIStateContext`** (`src/contexts/UIStateContext.tsx`) — already manages a collapsible *left* sidebar with `localStorage` persistence (`isSidebarCollapsed`, `toggleSidebarCollapsed`, key `zettelgarden-sidebar-collapsed`). A closable right rail is the same pattern: add `rightPaneOpen` + `toggleRightPane` + a persistence key. ~15 lines.
2. **`SidePanelLayout`** (`src/components/layout/SidePanelLayout.tsx`) — a generic, themed, closable side panel with a header + close button, already used by the pinned-card split view (`SplitViewLayout`). Its header/close affordance is the model for the rail's chrome.
3. **`Collapsible`** (`src/components/Collapsible.tsx`) — added in the simplification. Calm header + rotating chevron + optional count badge + `rightElement` slot + `aria-expanded`. Reuse for any collapsible rows inside tabs.
4. **Tab-strip pattern** — `ViewCardTabbedDisplay.tsx` (lines ~150-175) has the house style for tab strips: `<span>` buttons with `activeTab` state, blue underline on active, muted hover. Mirror this for the rail's tab switcher.
5. **`useIsMobile`** (`src/hooks/useIsMobile.ts`) — the 768px breakpoint hook. Mobile already has its own accordion shell (`ViewMobileLayout`); the rail work is desktop-first and must not regress mobile.

## Design decisions to resolve up front

These three need an answer before coding. My recommendation is marked **(rec)**; the next agent should confirm or adjust with the user.

### 1. Separate route vs. URL query param vs. in-pane only

Putting links on a separate page (`/app/card/:id/links`) breaks the live-adjacent feel that makes Obsidian's backlinks useful. **(rec)** Keep everything in the closable pane, but reflect the active tab in a query param (`?pane=links|metadata|entities`) so a view is shareable/bookmarkable without inventing a route. In-pane-only is the minimum viable; the query param is a nice-to-have in the last PR.

### 2. Composition with the pin feature

The app already opens a pinned card in a second side panel (`SplitViewLayout` → `SidePanelLayout`). If both the rail and a pinned card are open, you get four columns (nav · main · rail · pinned). **(rec)** Opening a pinned card **collapses the right rail automatically**, and closing the pinned card restores it. Decide this in PR 1, before either feature gets more complex.

### 3. The "add child" / "add backlink" affordances when the rail is closed

Today these live with their lists in the main column. When Children + References move to the Links tab, the `+` buttons move with them — but you still need to add a child/backlink with the rail closed. **(rec)** Keep a small, always-visible affordance in the body footer (e.g. a muted "＋ Child" / "＋ Link" text button row under the body), which auto-opens the rail to the Links tab when clicked. Alternative: show the affordance only when the rail is closed.

## Target layout

```
┌──────────┬────────────────────────────┬──────────────┐
│  Sidebar │  [breadcrumb] Title        │ Links│Meta│Ent│ ← tab strip + ✕
│          │  ★ 📌  [Edit] ⋮             │              │
│          │  Normal Summary Analysis   │ Children (3) │
│          │ ─────────────────────────  │ …            │
│          │                            │ Linked ref(7)│
│          │   Card body prose          │ …            │
│          │   (pure content — no       │ Parent / sib │
│          │    children/refs inline)   │ Related (2)  │
│          │                            │              │
│          │   ＋ Child   ＋ Link       │              │ ← footer affordance
│          │   ───────────────────────  │              │
│          │   Tasks · Entities/Files…  │              │
└──────────┴────────────────────────────┴──────────────┘
```

When the rail is closed, the main column reclaims full width; only the footer `＋` affordances remain as the way back in.

### Tab contents

| Tab | Contents (moved from where) |
|---|---|
| **Links** | Children + Linked references (two-way/incoming/outgoing) + BacklinkInput + Related cards + Parent/sibling nav |
| **Metadata** | Tags, Details (created/updated/link), Source article, Structured data |
| **Entities** | Linked entities |

After the move, the **main column** keeps only: `CardBody`, Tasks, and `ViewCardTabbedDisplay` (entities/files/history/summaries). The collapsible "Linked references" block and the Children section leave the main column entirely.

## Phased rollout

Four PRs, each independently shippable and reviewable. Match the cadence and discipline of the prior simplification PRs (single concern per commit, run `tsc --noEmit` + `npx vitest run` before each, preserve single-quote JS literals / double-quote JSX attributes, push each PR).

### PR 1 — Closable rail (no content changes)

**Goal:** establish open/close behavior in the context. Zero content moves yet.

- `UIStateContext`: add `rightPaneOpen` (default `true`), `setRightPaneOpen`, `toggleRightPane`; persist under key `zettelgarden-right-pane-open` (mirror the left-sidebar pattern, including the `getInitialState()` localStorage helper).
- `ViewPageHeader`: add a quiet toggle button in the action row (next to star/pin) that calls `toggleRightPane`. Icon: a simple panel/sidebar-right SVG. Title attr "Toggle info pane".
- `ViewPage` desktop layout: when `!rightPaneOpen`, render only the main column at full width (drop the `md:flex-row` wrapper, drop `ViewPageSidePanels`). When open, current two-column layout.
- **Decision #2 wiring:** in `UIStateContext` (or in `MainApp`/`SplitViewLayout`), when `pinnedCard` becomes non-null set `rightPaneOpen=false`; when pinned clears, restore prior state. Decide the cleanest place — likely a small `useEffect` in the component that sets the pinned card.
- **Mobile:** no change. `ViewMobileLayout` has its own shell; the toggle only renders on desktop (`hidden md:inline-flex`).

**Files:** `src/contexts/UIStateContext.tsx`, `src/components/cards/ViewPageHeader.tsx`, `src/pages/cards/ViewPage.tsx`, possibly `src/components/cards/SplitViewLayout.tsx` or `src/pages/MainApp.tsx`.

**Tests:** add a `UIStateContext` test asserting `rightPaneOpen` toggles and persists; assert the header toggle button is present in `ViewPageHeader`.

### PR 2 — Tabbed rail (no content moves yet)

**Goal:** refactor `ViewPageSidePanels` into a 3-tab pane, redistributing its *current* contents.

- New component `ViewPageSidePanelTabs` (or refactor in place): a tab strip (copy the `ViewCardTabbedDisplay` style) with state `activeTab: 'links' | 'metadata' | 'entities'`, plus a close `✕` button on the right of the strip.
- **Links tab** (initially): Parent + sibling nav only (the rest arrives in PR 3).
- **Metadata tab:** Tags, Details, Source article, Structured data (moved verbatim from current `ViewPageSidePanels`).
- **Entities tab:** Linked entities (moved verbatim).
- Keep all the existing prop-passing; this is purely a re-shelling of `ViewPageSidePanels`. Rename or wrap — but don't break the call site in `ViewPage`.
- **Mobile:** still no change. Mobile's accordions already cover this.

**Files:** `src/components/cards/ViewPageSidePanels.tsx` (refactor or split), `src/pages/cards/ViewPage.tsx` (call site).

**Tests:** render each tab, assert the expected sections appear and the others don't; assert tab switching works.

### PR 3 — Move Children + Linked references into the Links tab

**Goal:** the main column becomes pure content. This is the highest-impact, highest-risk PR.

- Move the Children section (lines ~117-145 of `ViewCardContentSection.tsx`) and the collapsible "Linked references" block + `BacklinkInput` into the rail's Links tab.
- The Links tab now holds: Parent/sibling nav, Children (+ sort), Linked references collapsible (+ sort), BacklinkInput, Related cards.
- **Main column** keeps: `CardBody`, Tasks, `ViewCardTabbedDisplay`. Add the **footer affordance** (Decision #3): a muted `＋ Child` / `＋ Link` row under the body that, when clicked, sets `rightPaneOpen=true` and `activeTab='links'`. The rail's active tab therefore needs to lift into `UIStateContext` too (add `rightPaneTab` + setter, or keep local and expose an imperative open — lifting is cleaner).
- Handle the empty states: a card with no children and no references should show a calm hint in the Links tab, not a wall of "No X yet."
- **Mobile:** the mobile accordions already have a "Navigation" section; decide whether Children + References move into the mobile accordions too (probably yes, for parity), but that can be a follow-up if it risks scope creep. Note it in the PR description either way.

**Files:** `src/components/cards/ViewCardContentSection.tsx`, `src/components/cards/ViewPageSidePanels.tsx` (Links tab), `src/contexts/UIStateContext.tsx` (if lifting `rightPaneTab`), `src/pages/cards/ViewPage.tsx`.

**Tests:** assert Children + References render in the rail, not the main column; assert the footer `＋` buttons open the rail to the Links tab; assert main column still renders body/tasks/tabbed-display.

### PR 4 — Polish: URL param + defaults + empty states

**Goal:** shareability and fit-and-finish. Lower priority; can defer.

- Reflect `activeTab` in `?pane=` via `useSearchParams`. On mount, read it back. Don't add a route.
- Sensible default tab: Links if the card has children/references, else Metadata.
- Pass over all empty states in the rail for tone (match the calm Obsidian style established in the simplification).
- Keyboard: a shortcut to toggle the rail (e.g. `Cmd/Ctrl-\`), following whatever convention `useKeyboardShortcuts` already uses.

**Files:** `src/components/cards/ViewPageSidePanels.tsx`, `src/hooks/useKeyboardShortcuts.ts`, possibly a small `useRightPaneTab` hook.

## Testing approach

- **Unit (Vitest + RTL):** context state for `rightPaneOpen`/`rightPaneTab`; header toggle button; tab switching; content present per tab; footer `＋` affordance opens rail.
- **Integration:** render `ViewPage` with a mocked `useViewPageContainer` and assert (a) rail closed → main column full width, (b) open Links tab → Children + References present, main column absent of them, (c) pinning a card collapses the rail.
- **Mobile regression:** the existing `ViewMobileLayout.test.tsx` must stay green untouched — mobile is out of scope and must not change behavior.
- Run `npx tsc --noEmit` and `npx vitest run` (full suite, currently 480 tests) before each PR. Baseline at handoff: 480 passing.

## Out of scope (file as follow-ups, don't bundle)

- **Mobile parity** for Children/References in the accordion shell — separate PR.
- **Side-metadata de-duplication** between `ViewPageSidePanels` (desktop) and `ViewMobileLayout` accordions — the two still duplicate ~150 lines with subtle differences (date formatting, entity descriptions). A shared component is worth a dedicated PR.
- **Resizable/draggable rail width** — Obsidian lets you drag the pane width. Nice but not now.
- **Multiple right-pane "views"** beyond tabs (e.g. outline, graph) — future feature, not part of this plan.

## Precise references for the next agent

- `UIStateContext` left-sidebar pattern to mirror: `src/contexts/UIStateContext.tsx` lines ~30-40 (interface), ~67-90 (state + persistence).
- Reusable closable panel: `src/components/layout/SidePanelLayout.tsx`.
- Reusable `Collapsible`: `src/components/Collapsible.tsx` (+ test `Collapsible.test.tsx`).
- Tab-strip house style: `src/components/cards/ViewCardTabbedDisplay.tsx` ~lines 150-175.
- Current right rail contents (to redistribute): `src/components/cards/ViewPageSidePanels.tsx` — Parent (L52), Source article (L92), Linked entities (L119), Tags (L168), Details (L~200).
- Current main-column Children + References to move: `src/components/cards/ViewCardContentSection.tsx` — Children section (~L117), Linked references `Collapsible` (~L140).
- `viewMode` definition: `src/pages/cards/ViewPageContainer.tsx` L37.
- App shell + pin composition: `src/pages/MainApp.tsx` ~L79-105, `src/components/cards/SplitViewLayout.tsx`.
- Mobile shell (do not regress): `src/components/cards/ViewMobileLayout.tsx` (+ `ViewMobileLayout.test.tsx`).
