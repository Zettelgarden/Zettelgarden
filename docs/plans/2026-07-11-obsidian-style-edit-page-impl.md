# Obsidian-style Edit Page — Implementation Plan

**Date:** 2026-07-11
**Status:** Proposed
**Goal:** Make `EditPage` match the closable, tabbed, header-driven style just established for `ViewPage`.
**Tracking:** file a bd issue before starting

## Current state of EditPage (verified)

`src/pages/cards/EditPage.tsx` (~478 lines) — `EditPage` wraps providers → `EditPageContent`. Routed at `card/:id/edit` and `card/new` (`src/pages/AppRoutes.tsx:84-85`).

Layout today:
- **`MobileTopBar`** (L304) — mobile only.
- **`EditorToolbar`** (L311) — a standalone bar reading `Editing: [card_id] - title` with an overflow menu (Process Entities & Facts, Save as Template, Delete Card). This is the closest thing to a header, but it's visually heavy and separate from the title.
- **Two-column body** (`flex flex-col md:flex-row gap-4 px-4`, L325):
  - **Main column** `md:w-2/3` (L326): `CardEditor` (title input, markdown toolbar, body textarea, Save/Cancel buttons, template picker, messages) → a stray double `<hr>` (L340, L342) → "Link" source-URL input + fill-from-URL button (L345) → Files list (`FileListItem`s, L375).
  - **Side panel** `md:w-1/3` (L391): `CardMetadata` (Card ID + helpers, Tags, References/backlink input, Details created/updated) → `CardSchemaSection` (structured data).

Supporting components (all have passing tests; 59 tests across the three):
- `CardEditor.tsx` — owns the title input, body, markdown toolbar, **Save/Cancel buttons (L186-187)**, dialogs.
- `CardMetadata.tsx` — Card ID, Tags, References, Details. Self-contained card (`bg-white rounded-lg p-4 shadow-sm`).
- `EditorToolbar.tsx` — the overflow menu + "Editing:" label.

**No mobile-specific edit layout exists** — the two-column just stacks. Out of scope to change.

## The ViewPage style to match (already shipped)

- **`ViewPageHeader`** — Obsidian-style: breadcrumb `[card_id]` + large title + quiet action-icon row (star/pin/rail-toggle) + overflow menu + segmented view-mode control. Title is static `text-2xl font-semibold`.
- **Main column** — pure content (body, tasks, footer affordances).
- **`ViewPageSidePanels`** — closable, tabbed rail (Links/Metadata/Entities): tab strip + `✕`, reads `rightPaneOpen`/`toggleRightPane`/`rightPaneTab` from `UIStateContext`, syncs `?pane=` via `useSearchParams`, smart default tab, empty-state tone at `text-gray-400`.
- **Global** — `Cmd/Ctrl-\` toggles the rail (`useKeyboardShortcuts`, mounted in `Sidebar`), so it already works anywhere the rail reads context state.

## Existing infrastructure to reuse (do NOT reinvent)

1. **`UIStateContext`** already owns `rightPaneOpen`, `setRightPaneOpen`, `toggleRightPane`, `rightPaneTab`, `setRightPaneTab` — with localStorage persistence + pin auto-collapse. EditPage reads the same state; no new context work.
2. **`ViewPageHeader`** — the visual template to clone for `EditPageHeader` (breadcrumb + title + action row + overflow). The `Menu` overflow pattern and quiet-icon styling are the house style.
3. **`ViewPageSidePanels`** — the tab-strip + `✕` + `?pane=` sync pattern. The URL-sync logic is currently inline there (~L82-103); **extract it into a `useRightPaneTab` hook** so EditPage can share it without duplication (this hook was flagged as a nice-to-have in the prior plan).
4. **`Collapsible`** — for any collapsible rows in the edit rail.
5. **Editor contexts** (`CardEditorProvider`/`EditorUIProvider`/`EditorMessagesContext`) — keep as-is; they're orthogonal to layout.

## Design decisions to resolve up front

### 1. Title: editable input in the header, or static title + input in body? **(rec) Editable input in the header.**
The ViewPageHeader title is static text. For editing, put the `<input>` directly in the header, styled to read as a title (`text-2xl font-semibold`, borderless, full-width on focus). This is the closest visual match and keeps "title is the primary editable thing" obvious. The suggest-title button (currently inside `CardEditor`) moves up next to the input.

### 2. Save/Cancel: header or bottom of body? **(rec) Header action row.**
Move Save/Cancel into the header's action row (where ViewPageHeader has Edit/overflow). They're then always reachable when the body is long. Remove the duplicate Save/Cancel block at the bottom of `CardEditor` (L186-187). Keep a sticky/floating consideration out of scope — header placement is enough.

### 3. Rail tab structure for editing? **(rec) Two tabs: Metadata + Links.**
Mirror ViewPage's pattern but trimmed to what editing needs:
- **Metadata** — Card ID (+ helpers), Tags, Source/Link, Schema, Details (created/updated).
- **Links** — References: the backlink input (`BacklinkInputDropdownList`) that appends `[[id|*|]]` to the body. (Showing read-only children/outgoing refs is a possible enhancement but scope creep — note as follow-up.)

Files stay in the **main column** (they're directly editable/attachable during editing, unlike view mode). The stray double `<hr>` (L340/L342) gets removed.

### 4. Share `?pane=` URL sync with ViewPage? **(rec) Yes, via a new `useRightPaneTab` hook.**
Extract the mount-read + write-on-change logic from `ViewPageSidePanels` into `src/hooks/useRightPaneTab.ts`. Both pages call it; the hook owns `?pane=` so a deep link to a tab works on either page. EditPage's rail reuses `rightPaneTab`/`setRightPaneTab` from context just like ViewPage.

### 5. Mobile? **(rec) No new mobile edit layout.**
EditPage has none today and the stacked two-column works. The new header must be responsive (`hidden md:` on desktop-only bits, like ViewPageHeader). The rail toggle button is desktop-only. Mobile keeps the current stacked editor + metadata.

---

## Phased rollout

Three PRs, each independently shippable. Run `npx tsc --noEmit` + `npx vitest run` before each. Baseline: 512 tests passing, tsc clean.

### PR 1 — Obsidian-style header (no rail changes)

**Goal:** replace `EditorToolbar` with a `ViewPageHeader`-style header. Body layout untouched.

- New `src/components/cards/EditPageHeader.tsx`:
  - Breadcrumb `[card_id]` (font-mono, muted) — for new cards show `[new]` or the proposed id.
  - **Editable title input** styled as a title (`text-2xl font-semibold`, borderless, `focus:ring` on focus) + the suggest-title button inline.
  - Action row: **Save** (primary) + **Cancel** (outline) — moved here from `CardEditor`'s bottom — + overflow `Menu` (Process Entities & Facts checkbox, Save as Template, Delete Card) lifted from `EditorToolbar`.
  - Desktop-only rail toggle button (`hidden md:inline-flex`, `title="Toggle info pane"`) reading `toggleRightPane`/`rightPaneOpen` — same SVG as `ViewPageHeader`. (The rail isn't wired to anything new yet, but the toggle is harmless and sets up PR 2.)
- `EditPage.tsx`: swap `<EditorToolbar>` (L311) for `<EditPageHeader>`. Pass through the handlers `EditorToolbar` already received plus `handleSaveCard`/`handleCancelButtonClick`/`handleSuggestTitle`/title state.
- `CardEditor.tsx`: **remove** the Save/Cancel block (L186-187) and the title input + suggest button (they live in the header now). `CardEditor` becomes body-only: messages, template picker, markdown toolbar, body textarea, dialogs. Drop the now-unused props.
- Delete `EditorToolbar.tsx` + `EditorToolbar.test.tsx` (its menu items move to the header) — or keep `EditorToolbar` as just the menu if extraction is cleaner. **(rec)** fold the menu into `EditPageHeader` and delete `EditorToolbar`.

**Files:** new `EditPageHeader.tsx` (+ test), `EditPage.tsx`, `CardEditor.tsx` (+ test update), delete `EditorToolbar.tsx`/`.test.tsx`.

**Tests:** `EditPageHeader` renders title input, Save/Cancel, overflow items; suggest button calls handler; Save calls handler. Update `CardEditor.test.tsx` to drop title/save-cancel assertions that moved out.

**Gate:** tsc clean, full suite green.

### PR 2 — Extract `useRightPaneTab` + closable edit rail (no content moves yet)

**Goal:** the edit side panel becomes closable, reusing context state; the `?pane=` sync is shared.

- New `src/hooks/useRightPaneTab.ts`: extract the two `useEffect`s from `ViewPageSidePanels` (mount-read `?pane=` with validation, write-on-change with `replace`). Signature roughly `useRightPaneTab({ hasRelationships }: { hasRelationships: boolean })` — returns nothing, just syncs `rightPaneTab` ↔ URL. Reads `rightPaneTab`/`setRightPaneTab` from `useUIState` internally.
- `ViewPageSidePanels`: replace its inline effects with the hook call (delete ~L82-103, add `useRightPaneTab({ hasRelationships: ... })`). Behavior identical — pure refactor.
- `EditPage.tsx`: wrap the existing `md:w-1/3` block (L391-414) so it's conditionally rendered on `rightPaneOpen`, and the main column reclaims full width when closed (copy the `ViewPage` pattern: `${rightPaneOpen ? 'md:w-2/3' : 'w-full'}`). Add the tab strip + `✕` header chrome around `CardMetadata` + `CardSchemaSection` (single tab for now — "Metadata" — tabbed split is PR 3) OR skip tabs here and just add the `✕` close + call `useRightPaneTab`. **(rec)** add the full tab strip now (single Metadata tab) so PR 3 only reshuffles content.
- Call `useRightPaneTab` in the edit rail too.

**Files:** new `useRightPaneTab.ts` (+ test), `ViewPageSidePanels.tsx`, `EditPage.tsx`.

**Tests:** `useRightPaneTab` test (mount-read valid/invalid, write-on-change, smart default). `ViewPageSidePanels` URL tests should stay green unchanged (they now exercise the hook). Add an EditPage-level assertion that the rail hides when `rightPaneOpen` is false.

**Gate:** tsc clean, full suite green. No behavior change to ViewPage.

### PR 3 — Tabbed edit rail + content moves

**Goal:** split the edit rail into Metadata/Links tabs; move Source/Link input out of the main column.

- Edit rail becomes two tabs (strip modeled on `ViewPageSidePanels`):
  - **Metadata**: Card ID (+ helpers), Tags, Source/Link input + fill button, Schema, Details.
  - **Links**: References — `BacklinkInputDropdownList` (the backlink input currently in `CardMetadata`).
- Move the "Link" section (L345-389 in `EditPage`) and the `CardMetadata` References block into the rail. `CardMetadata` is refactored to render its pieces per-tab (or split into `EditMetadataTab`/`EditLinksTab` sub-components — **rec**: keep `CardMetadata` but accept a `tab` prop, lowest churn).
- **Main column** becomes pure editor: header (title + actions), template picker (new cards), markdown toolbar, body textarea. Files list stays (directly editable). Remove the stray double `<hr>`.
- Smart default tab: Metadata (an editor almost always wants Card ID/Tags visible first).

**Files:** `EditPage.tsx`, `CardMetadata.tsx` (+ test), `EditPageHeader.tsx` if any prop plumbing.

**Tests:** assert Source/Link and References render in the rail, not the main column; tab switching; Files still in main column.

**Gate:** tsc clean, full suite green.

---

## Out of scope (file as follow-ups, don't bundle)

- **Mobile edit layout** — no accordion shell for editing today; the stacked two-column is acceptable.
- **Read-only children/references in the edit Links tab** — `fetchCard` already pulls them; showing them read-only while editing is a nice enhancement but separate.
- **Sticky/floating Save bar** — header placement is enough for now.
- **De-duplicating `CardMetadata` (edit) with the view rail's Metadata tab** — they overlap (Tags, Details) but differ enough (editable vs read-only, Card ID editor) to warrant a dedicated pass.

## Risk notes

- **`EditorToolbar` deletion** (PR 1) is the riskiest step — confirm nothing else imports it. At time of writing only `EditPage.tsx` does.
- **Title-as-input in the header** (PR 1) changes a familiar affordance; keep the input's styling obviously editable (focus ring) so users don't think it's static.
- **`useRightPaneTab` extraction** (PR 2) must preserve ViewPage's exact behavior — the existing `ViewPageSidePanels` URL tests are the regression net; they must stay green untouched.
- **Shared `rightPaneTab` state across view/edit** — toggling/switching a tab on one page persists to the other. This is desirable (consistency) but worth a sentence in the PR description.
