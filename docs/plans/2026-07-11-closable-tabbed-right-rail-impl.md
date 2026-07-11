# Closable, Tabbed Right Rail — Implementation Plan

**Date:** 2026-07-11
**Status:** Ready to build
**Design doc:** `docs/plans/2026-07-11-closable-tabbed-right-rail-design.md`
**Tracking:** bd issue `Zettelgarden-5wp`

This is the execution plan. The design doc is the *what/why*; this is the *how*, with exact edits keyed to the current tree. Code references below were all verified against the working copy on 2026-07-11.

## Verification of the design doc's assumptions

All claims in the design doc's "Existing infrastructure" and "Precise references" sections check out:

- **Baseline:** `npx vitest run` → 45 files, **480 passing**. `npx tsc --noEmit` clean. Matches the handoff baseline.
- **`UIStateContext`** (`src/contexts/UIStateContext.tsx`) left-sidebar pattern exists exactly as described: `SIDEBAR_COLLAPSED_KEY = 'zettelgarden-sidebar-collapsed'`, `getInitialSidebarState()`, `setIsSidebarCollapsed`, `toggleSidebarCollapsed`. The right-pane work mirrors this verbatim.
- **`ViewPageSidePanels.tsx`** (220 lines) section order confirmed: Parent (L52) → Source Article (L92) → Linked Entities (L119) → Related Cards (L151) → Structured Data (L159) → Tags (L168) → Details (border-t block at the end).
- **`ViewCardContentSection.tsx`** (227 lines) confirmed: `CardBody` (L108) → Children section (L119) → `Collapsible` "Linked references" (L144) → Tasks (L205) → `ViewCardTabbedDisplay` (L217).
- **`ViewPage.tsx`** desktop layout confirmed: `flex flex-col md:flex-row gap-4` wrapper, main `flex-1 md:w-2/3`, then `<ViewPageSidePanels>` always rendered.
- **`ViewPageHeader.tsx`** action row confirmed: star → pin → Edit → overflow menu. The rail toggle slots in next to star/pin.
- **Tab-strip house style** confirmed at `ViewCardTabbedDisplay.tsx` ~L150-175 (`<span>` buttons, `activeTab` state, `border-b-2 border-blue-600` on active, muted hover).
- **`Collapsible`** (`src/components/Collapsible.tsx`) API confirmed: `title`, `count`, `defaultOpen`, `rightElement`, `idSuffix`, `aria-expanded`.
- **`SidePanelLayout`** (`src/components/layout/SidePanelLayout.tsx`) is the reusable closable panel with `onClose` + themed header.
- **`useKeyboardShortcuts`** exists with a colocated test — the PR 4 shortcut goes here.
- **`useIsMobile`** (768px breakpoint) confirmed; mobile path is `ViewMobileLayout` and is out of scope.

## Resolved decisions

The design doc left three decisions open. Decided answers:

### Decision #1 — In-pane only (PR 4 adds query param)
Keep the rail in the main card view; **no new route**. Add `?pane=links|metadata|entities` as a PR-4 nicety via `useSearchParams`. Default to in-pane state in PRs 1–3.

### Decision #2 — Pin auto-collapses the rail, restore on unpin (do it in PR 1)
When `pinnedCard` becomes non-null, collapse the rail; when it clears, restore the prior value. **Wiring location: inside `UIStateContext`**, not scattered across call sites — all pin mutations funnel through the single `setPinnedCard` there (`ViewPageContainer.tsx:213` sets it, `SplitViewLayout.tsx:21` and `ViewPageContainer.tsx:210` clear it). A `useEffect` watching `pinnedCard` transitions keeps the logic in one testable place.

### Decision #3 — Body-footer `＋ Child` / `＋ Link` affordance (PR 3)
Add a muted row under the body, **always visible**, that opens the rail to the Links tab (`rightPaneOpen=true`, `rightPaneTab='links'`). Always-visible beats "only when closed" because it's the single, predictable entry point and doubles as a count-free summary affordance.

This means the rail's **active tab must be lifted into `UIStateContext`** (`rightPaneTab`) in PR 3, not kept local. Do the lift in PR 3 alongside the footer.

---

## PR 1 — Closable rail (no content moves)

**Goal:** open/close behavior, persisted, with pin auto-collapse. Zero content changes.

### Edits

**`src/contexts/UIStateContext.tsx`** — add right-pane state mirroring the sidebar pattern:
- Add `const RIGHT_PANE_OPEN_KEY = 'zettelgarden-right-pane-open';`
- Add `const getInitialRightPaneState = (): boolean => { … return stored !== 'false'; }` (default **open**).
- Add interface fields: `rightPaneOpen: boolean`, `setRightPaneOpen: (open: boolean) => void`, `toggleRightPane: () => void`.
- Add state `const [rightPaneOpen, setRightPaneOpenState] = useState<boolean>(getInitialRightPaneState);` plus setters that persist (copy `setIsSidebarCollapsed` / `toggleSidebarCollapsed`).
- **Decision #2 wiring:** add a ref `const priorRightPaneOpen = useRef(rightPaneOpen);` and a `useEffect` on `[pinnedCard]`: when `pinnedCard` goes non-null, stash `priorRightPaneOpen.current = rightPaneOpen` then `setRightPaneOpenState(false)`; when it goes back to null, `setRightPaneOpenState(priorRightPaneOpen.current)`. Do **not** persist the forced-closed value to localStorage (only the user toggle persists), so wrap the auto-close in `setRightPaneOpenState` directly, not the public setter.
- Export all three new fields on the provider value.

**`src/components/cards/ViewPageHeader.tsx`** — add a quiet toggle button in the action row, between pin and Edit:
```tsx
<button
  type="button"
  onClick={toggleRightPane}
  title="Toggle info pane"
  className={`p-2 rounded-md transition-colors ${
    rightPaneOpen ? 'text-gray-700 hover:bg-gray-100' : 'text-gray-400 hover:text-gray-700 hover:bg-gray-100'
  }`}
>
  {/* sidebar-right SVG */}
</button>
```
Pull `rightPaneOpen` + `toggleRightPane` from `useUIState()`. Use a simple `M4 4h16v16H4z`-style panel-right icon (chevron toward the right edge).

**`src/pages/cards/ViewPage.tsx`** — gate the rail on `rightPaneOpen`:
```tsx
const { toggleMobileSidebar, setPinnedCard, rightPaneOpen } = useUIState();
…
<div className="flex flex-col md:flex-row gap-4">
  <div className={`space-y-4 overflow-y-auto ${rightPaneOpen ? 'flex-1 md:w-2/3' : 'w-full'}`}>
    {/* main column */}
  </div>
  {rightPaneOpen && <ViewPageSidePanels … />}
</div>
```
Note the wrapper `md:flex-row` can stay; just conditionally render the panel and widen the main column when absent.

**Mobile:** the toggle button gets `hidden md:inline-flex` so it never renders on mobile. No `ViewMobileLayout` changes.

### Tests
- New `src/contexts/UIStateContext.test.tsx` (none exists today): render provider, assert `rightPaneOpen` starts `true`, `toggleRightPane()` flips it, persists to localStorage key, survives remount; assert pinning a card forces `rightPaneOpen=false` and unpinning restores.
- Extend an existing `ViewPageHeader` render test (or add one) asserting the toggle button is present with `title="Toggle info pane"`.

### Gate
`tsc --noEmit` clean, `vitest run` green, no mobile test changes.

---

## PR 2 — Tabbed rail (no content moves)

**Goal:** re-shell `ViewPageSidePanels` into 3 tabs using its *current* contents.

### Edits

**`src/components/cards/ViewPageSidePanels.tsx`** — keep the export name and props (so `ViewPage` call site is untouched). Internally:
- Add local state `const [activeTab, setActiveTab] = useState<'links' | 'metadata' | 'entities'>('metadata');` (Metadata is the densest today; Links is near-empty until PR 3).
- Render a tab strip at the top (copy `ViewCardTabbedDisplay` style: `<span>` buttons, `border-b-2 border-blue-600` active, muted hover) plus a `✕` button on the right that calls `toggleRightPane` (pull from `useUIState`).
- Redistribute **existing** sections:
  - **Links tab:** Parent + sibling nav only (the L52 block).
  - **Metadata tab:** Source Article (L92), Structured Data (L159), Tags (L168), Details (border-t block).
  - **Entities tab:** Linked Entities (L119). Leave `RelatedCards` (L151) in Metadata for now — it moves to Links in PR 3.
- Wrap the scrollable body so the strip stays pinned: `<div className="md:w-1/3 flex flex-col">` with the strip in a non-scrolling header and tab content in `overflow-y-auto`.

**`src/pages/cards/ViewPage.tsx`** — no change (props unchanged). The only consumer adjustment is that `ViewPageSidePanels` now reads `toggleRightPane` itself.

### Tests
- Render `ViewPageSidePanels` with sample props; assert default tab shows Tags/Details; click "Links" → Parent nav appears, Tags disappear; click "Entities" → Linked Entities appear; assert `✕` calls `toggleRightPane` (mock context).

### Gate
`tsc --noEmit` clean, full suite green. This PR is a pure re-shell — visually different, behaviorally equivalent.

---

## PR 3 — Move Children + Linked references into the Links tab

**Goal:** main column becomes pure content. Highest-impact, highest-risk PR.

### Edits

**`src/contexts/UIStateContext.tsx`** — lift the active tab (Decision #3):
- Add `rightPaneTab: 'links' | 'metadata' | 'entities'` + `setRightPaneTab`. Default `'links'` (most cards have some relationship data; PR 4 will make the default smart). No persistence yet (PR 4).

**`src/components/cards/ViewCardContentSection.tsx`** — **remove** the Children section (L119–143) and the `Collapsible` Linked references block (L144–200, including `BacklinkInput`). Main column keeps `CardBody`, Tasks, `ViewCardTabbedDisplay`. Props `onCreateChildCard`, `categorizedReferences`, `onAddBacklink` become unused here — remove them from this component's interface and from the `ViewPage` call site, and pass them through to `ViewPageSidePanels` instead.
- **Add the footer affordance** (Decision #3) directly under the `CardBody` block, before Tasks:
```tsx
<div className="flex items-center gap-4 -mt-2 text-xs text-gray-400">
  <button onClick={() => { setRightPaneOpen(true); setRightPaneTab('links'); onCreateChildCard(); }}
    className="hover:text-blue-600 transition-colors">＋ Child</button>
  <button onClick={() => { setRightPaneOpen(true); setRightPaneTab('links'); }}
    className="hover:text-blue-600 transition-colors">＋ Link</button>
</div>
```
Pull `setRightPaneOpen`/`setRightPaneTab` from `useUIState`.

**`src/components/cards/ViewPageSidePanels.tsx`** — Links tab now holds: Parent/sibling nav, **Children section** (with its `SortControl` and `onCreateChildCard`), **`Collapsible` Linked references** (bidirectional/incoming/outgoing + `SortControl` + `BacklinkInput`), **Related Cards**. This means new props flow into `ViewPageSidePanels`: `viewingCard`, `categorizedReferences`, `onCreateChildCard`, `onAddBacklink`, `children`/references sort state (lift the two `useState<SortMethod>` from `ViewCardContentSection` into the Links tab, or keep them local in the Links tab subcomponent).
- Switch the tab strip from local `activeTab` to the lifted `rightPaneTab`/`setRightPaneTab`.
- Empty states: a card with no children **and** no references shows one calm line ("No links yet") in the Links tab, not three "No X yet" blocks.

**`src/pages/cards/ViewPage.tsx`** — repoint props: `onCreateChildCard`, `categorizedReferences`, `onAddBacklink` move from the `ViewCardContentSection` call to the `ViewPageSidePanels` call. `ViewCardContentSection` loses those three props.

### Tests
- Assert `ViewCardContentSection` no longer renders "Children" or "Linked references" labels; still renders body/Tasks/tabbed-display.
- Assert the footer `＋ Child`/`＋ Link` buttons set `rightPaneOpen=true` + `rightPaneTab='links'` (mock context).
- Assert `ViewPageSidePanels` Links tab renders Children + Linked references + BacklinkInput + Related Cards.
- Integration: render `ViewPage` desktop with a mocked `useViewPageContainer` returning a card with children+refs → assert they appear in the rail, not the main column.

### Gate
Full suite green. **Watch the mobile regression:** `ViewMobileLayout.test.tsx` must stay green untouched — Children/References stay in the mobile accordion for now (noted as out of scope).

---

## PR 4 — Polish (defer-safe)

- `?pane=` query param via `useSearchParams` in `ViewPageSidePanels`; read on mount, update on tab change.
- Smart default tab: `links` if card has children/references, else `metadata`. Compute once in the Links-tab mount.
- Persist `rightPaneTab` to localStorage (optional — match the sidebar discipline).
- Keyboard shortcut to toggle the rail in `useKeyboardShortcuts.ts` (e.g. `Cmd/Ctrl-\`), following the hook's existing convention; add a test there.
- Empty-state tone pass across all three tabs (calm Obsidian style).

---

## Out of scope (file as follow-up bd issues, do not bundle)

- Mobile parity: Children/References in `ViewMobileLayout` accordions.
- De-duplicate the ~150 lines of side-metadata shared between `ViewPageSidePanels` and `ViewMobileLayout` into one component.
- Resizable/draggable rail width.
- Additional rail "views" (outline, graph).

## Risk notes for the implementer

- **PR 3 prop plumbing** is the fiddly part: three props (`onCreateChildCard`, `categorizedReferences`, `onAddBacklink`) and two sort states move across components. Do it in one commit, run the full suite immediately after.
- **`ViewPageSidePanels` props change** in PR 3 — there's exactly one call site (`ViewPage.tsx`), so it's contained, but the `RelatedCards` callbacks (`onRelatedCardClick`, `onRelatedCardAddReference`) already live there and stay.
- **Pin auto-collapse** (PR 1) must not persist the forced-closed state to localStorage, or reloading while a card is pinned would silently overwrite the user's saved preference. Use the raw state setter for the auto-close, the public (persisting) setter only for the explicit toggle.
