# Frontend Component Consistency Analysis & Plan

Date: 2026-08-09
Scope: `zettelkasten-front/src/components` (216 non-test/non-example TSX components)

> Note: this document and its concrete numbers were independently reviewed by a
> code-review subagent (`reviewer`) against the actual source on 2026-08-09.
> Corrections from that review are incorporated inline below (marked `review:`).

## Summary of findings

The frontend has **no component primitives layer**. Components are hand-written
with repeated inline Tailwind markup. Two styling approaches coexist — raw
Tailwind and Headless UI — but even the Headless UI components re-implement the
shell (overlay, sizing, button styles) per file rather than through a shared
primitive. This produces real, observable duplication and drift across the app.

There are no fully duplicated identical pages, **but** there are many
near-duplicate *patterns* (confirm dialogs, menus, badges, top bars, dialog
shells, spinners) that are each re-implemented 2–6 times with subtly different
props, styling, and behavior. That is the material problem.

---

## Evidence

### 1. Buttons: shared `Button.tsx` exists but is largely ignored

- `Button.tsx` is a real shared primitive (`variant`/`size`) — imported by **25**
  files (`review: doc originally said 24; 25 verified`). It is the **only** place
  `bg-palette-dark` is used as a primary button.
- The flip side is understated: **112** component/page files contain a raw
  `<button>` with styling classes and no `Button` import (`review: much more than
  the original "50+"; distribution: tasks 23, cards 15, rss 12, admin 8, schemas
  7, entities 7, search 6, + 37 others`).
- Primary-button color is inconsistent across the app:
  - `bg-blue-600`/`700`/`500` — the dominant ad-hoc choice (78/47/40 uses)
  - `bg-palette-dark` / `bg-palette-darkest` — the "brand" palette (3/2 uses)
  - `bg-indigo-600/700` — 2 uses
- So a user sees multiple distinct "primary" buttons depending on the screen,
  and the brand color is barely used at all.

### 2. Confirm dialogs: 4+ incompatible implementations

| File | Strategy | Severity/color prop | Extra behavior |
|---|---|---|---|
| `tasks/ConfirmDialog.tsx` | raw Tailwind | `variant` (default `danger`) | — (no Escape/scroll-lock) |
| `admin/ConfirmDialog.tsx` | raw Tailwind | `severity` (default `warning`) | escape key, scroll lock, checkbox, Promise hook `useConfirmDialog` — **dead code, 0 importers** (`review`) |
| `rss/RssConfirmDialog.tsx` | Headless UI | `dangerous: boolean` | animation |
| `tabs/RollbackConfirmDialog.tsx` | raw Tailwind (custom) | — | custom change-preview body |

All four render a modal with title/message/cancel/confirm, but with different
prop names (`variant` vs `severity` vs `dangerous`, `onClose` vs `onCancel`),
different visual treatment, and different a11y behavior (only `admin` locks
scroll / handles Escape). A consumer cannot predict the API or the look.

### 3. Dialog shell: raw vs Headless UI split, styling repeated

Of 29 `*Dialog.tsx` files, **11 are fully raw** (no Headless import) and **18 use
Headless UI `Dialog`** (`review: corrected from the original 14/15 split, which
miscounted files whose overlay divs contain `fixed inset-0``). Either way, no
shared `Modal` primitive exists — every shell (raw *and* Headless) re-types the
overlay/sizing classes. Entities illustrate the drift: `EntityDialog.tsx`
(`max-w-3xl rounded-2xl`), `AddEntityDialog.tsx` (`max-w-md rounded-lg`), and
`EditEntityDialog.tsx` (`max-w-2xl rounded-xl`) are three dialog shells in one
folder, none reusing a shared `Modal` (`review: sizes/radii verified`).

Additionally, hand-rolled modal shells also live *outside* `*Dialog.tsx`:
`EntityPage.tsx` (inline "Confirm Merge" dialog), `FactPage.tsx`, `SchemaPage.tsx`,
`ViewPageContainer.tsx`, `AdminPage.tsx`, `FileVault.tsx` all contain
`fixed inset-0` shells. The migration surface is larger than the 29 `*Dialog.tsx`
files alone (`review: missing from original doc`).

### 4. Menus/popovers: multiple ad-hoc implementations

`PopupMenu`, `CardListMenu`, `TaskDropdown`, `TaskListOptionsMenu`,
`SavedSearchesMenu`, `AddTagMenu`, `SearchTagMenu`, `SearchTagDropdown`,
`RemoveTagMenu`, `SidebarMobileMenu`, `QuickTagPopover`,
`CardBodyHelpPopover`, `BacklinkInputDropdownList`. Headless `Menu` is used in
some (`CardListMenu`, `TaskListOptionsMenu`, `AddTagMenu`, `SearchTagDropdown`)
and raw in others (`TaskDropdown`, `SavedSearchesMenu`, `RemoveTagMenu`,
`PopupMenu`, `SidebarMobileMenu`, `QuickTagPopover`, `SearchTagMenu`) — same job,
two approaches.

### 5. Badges: duplicated

`admin/StatusBadge.tsx` and `scheduler/JobStatusBadge.tsx` implement the same
colored pill with near-identical class maps (`bg-green-100 text-green-800`, etc.)
but different props and styling (`px-2 py-1 rounded` + `value/type/label` vs
`px-3 py-1 rounded-full` + `status` + dot/pulse). Any new "status" reinvents this.

### 6. Top bars / headers: partial duplication + dead code

`layout/MobileTopBar.tsx` (generic, documented, ~7 importers) vs
`rss/RssMobileTopBar.tsx` (raw, 1 importer) genuinely duplicate each other.
`review: AdminTopBar.tsx` is a **desktop** admin header with **zero importers**
(dead code) — delete it rather than migrate it.

### 7. Toast & spinner: shared primitives missing

- `toast/ToastContext.tsx` + `Toast.tsx` exist as a shared system (good).
- No `<Spinner/>` exists; spinner markup is hand-rolled everywhere. `review:
  scope correction` — 23 files contain spinner markup, but only **11** carry the
  exact standard Tailwind SVG (`d="M4 12a8 8 0 018-8V0"`); the other 12 use at
  least **3 different** spinner designs (`border-b-2 rounded-full` divs,
  `border-2 border-t-transparent` divs, a spinning unicode `⟳`, a rotating refresh
  icon). Core claim stands: no shared primitive, multiple inconsistent hand-rolled
  spinners.

### 8. Styling drift

- Brand palette (`palette.*`) appears only in `Button.tsx`, `Button.test.tsx`, and
  `SaveAsTemplateDialog.tsx` (3 mentions); `modern.*` has **zero** usages.
- Most UI uses raw gray/blue Bootstrappy classes.
- No `Field`/`Input`/`Label` primitive (only a specialized `BacklinkInput.tsx`).
- One CSS-module file (`ToggleSlider.module.css`) sits off the Tailwind pattern.

---

## Root causes

1. **No primitives layer / design-token story.** Tailwind is present, but there is
   no shared `Button`, `Modal`, `ConfirmDialog`, `Menu`, `Dropdown`, `Badge`,
   `Spinner`, `Input` abstraction that components are required to use.
2. **Two coexisting patterns.** Headless UI was adopted for some dialogs/menus but
   never as *the* mechanism, so raw and Headless implementations both exist.
3. **No review gate.** Nothing enforces "use the primitive." New features (RSS,
   admin, scheduler) each re-implemented the shell.
4. **Palette underused.** Two palettes defined (`palette`, `modern`) but ignored;
   the actual UI drifted to Tailwind's default blue/gray, burying the brand.

---

## Proposed plan (phased, low-risk)

Each phase is independently shippable; ordering is by leverage (biggest
consistency win first, lowest risk).

### Phase A — Define the primitives layer (foundation)
Create `src/components/ui/` with small, typed, tested primitives:
- `Button.tsx` (absorb/expose existing `Button.tsx`; **add a `danger` variant** —
  current variants are only `primary|secondary|outline`, `review`; preserve the
  existing consistent `min-h-[44px]` touch-target convention)
- `Modal.tsx` (single Headless-UI-based modal shell: overlay, sizing, Escape,
  focus trap, scroll-lock, dismiss, `aria-*`) — replaces the 11 raw `*Dialog`
  shells + the inline `fixed inset-0` shells in `EntityPage`/`FactPage`/`SchemaPage`/
  `ViewPageContainer`/`AdminPage`/`FileVault`, and standardizes the 18 Headless ones
- `ConfirmDialog.tsx` (built on `Modal`). **Baseline decision required (`review`):**
  the `admin` `useConfirmDialog` hook is currently dead code **and** has a real bug —
  it only calls `resolve()` in `onConfirm`, so cancelling leaves any
  `await confirm(...)` hanging forever. Either fix that cancel path (resolve
  `false` on cancel, add tests) or derive the API from `tasks/ConfirmDialog.tsx`
  (the one with 3 live consumers) and fold in admin's Escape/scroll-lock/checkbox
  features.
- `Menu.tsx` / `Dropdown.tsx` (single source built on Headless `Menu`)
- `Badge.tsx` (absorb `StatusBadge`/`JobStatusBadge`)
- `Spinner.tsx` (kill ALL hand-rolled spinner markup: SVG, border-div, unicode)
- `Field.tsx` / `Input.tsx` / `Select.tsx` / `Label.tsx` (form primitives)
- `DialogStateContext` compatibility: ensure the new `Modal`/`ConfirmDialog` still
  work with the global dialog-open state used by keyboard-shortcut dialogs (`review`)
- Decide a **single primary-button color** (proposal: `bg-palette-dark`, the brand)
  and delete the ad-hoc blue/indigo buttons as they're touched. **Decision (z11.4):
  `bg-palette-dark` (#38a3a5) confirmed; see `docs/ui-primitives-how-to-use.md`.**

Write a short `docs/` page: "UI primitives — how to use" (see
`docs/ui-primitives-how-to-use.md`), and add a Vitest spec per primitive.

### Phase B — Seed adoption on the highest-traffic surfaces
Migrate the most-visible/least-risk spots first to prove the pattern and show
immediate look-consistency:
- Button surfaces: `CardListItem`, `TaskListItem`, `EditPageHeader`, `ViewPageHeader`
  (`review correction`: `Header.tsx` is typography-only — `HeaderTop`/`H1`–`H6`, no
  buttons — and `Sidebar.tsx` has no buttons, so those aren't the right button targets)
- All confirm flows → `ConfirmDialog` (this unifies 4 implementations)
- Table headers (`TableComponents.tsx`) → keep, but route through `Button` where interactive

### Phase C — Sweep the remaining components
Migrate dialog shells → `Modal`, menus → `Menu`, status pills → `Badge`,
spinners → `Spinner`, form fields → `Field`/`Input`. Delete the superseded
`tasks/ConfirmDialog.tsx`, `rss/RssConfirmDialog.tsx`, `admin/StatusBadge.tsx`,
`admin/ConfirmDialog.tsx`, and `AdminTopBar.tsx` (the latter two are dead code
with zero importers — free wins that also shrink the migration surface, `review`)
as their call sites migrate / outright.

### Phase D — Governance
- **ESLint does not currently exist in this repo** (`review`): no config, no lint
  script; `build` is `tsc && vite build`. Adding ESLint is a greenfield tooling
  introduction, so start with a lightweight grep/CI check (e.g., block `fixed
  inset-0` outside `ui/`), and treat ESLint as a follow-up rather than the first gate.
- **Landed (Zettelgarden-z11.16):** `npm run check:primitives`
  (`scripts/check-ui-primitives.mjs`) fails on raw centered modal shells
  (`fixed inset-0` + `items-center` + `justify-center`) and raw Headless `Dialog`
  imports outside `src/components/ui/`; allows non-dialog `fixed inset-0`
  (bottom sheets, backdrops, popovers, overlays) and a small allowlist for
  FileVault's fullscreen/lightbox/upload overlays. Documented in
  `docs/ui-primitives-how-to-use.md`.
- Route new dialogs/menus/buttons through the primitives by convention documented
  in `docs/`.
- Optionally normalize to a single palette in `tailwind.config.js` (retire
  `modern.*` or commit to it) so brand tokens are used.

---

## Guardrails & risk

- **Expect a11y behavior to *change* (for the better) as a deliberate consequence.**
  Migrating the raw dialogs to a Headless-based `Modal` adds focus trapping,
  `aria-*`, and Escape handling they currently lack. This is an improvement, but it
  is technically a behavior change that can affect keyboard/e2e expectations — add
  the existing `e2e/smoke.ts` to the quality gates (`review`).
- Keep it mechanical: phase by component folder, run `npm run test:run`,
  `npm run format:check`, `npm run build`, and `e2e/smoke.ts` after each.
- The end goal is a **consistent look and a single place to change each UI
  primitive**, not a UI redesign. Visual changes should be minimal (except where
  the shared `Modal`/`Button` intentionally unifies look).
- ConfirmDialog: consolidate the 4 APIs into one, fixing the `useConfirmDialog`
  cancel-path bug; keep `RollbackConfirmDialog` as a thin specialization *on top
  of* the shared `ConfirmDialog`/`Modal`.
- **Dead code handling:** `admin/ConfirmDialog.tsx`, `AdminTopBar.tsx`, and
  `admin/StatusBadge.tsx` (0 importers) can be deleted outright in Phase C without
  call-site migration.

## Acceptance criteria

- [ ] `src/components/ui/*` primitives exist, typed, tested.
- [ ] No new raw `fixed inset-0` modal shells; existing ones migrated (incl. inline
      shells in `EntityPage`/`FactPage`/`SchemaPage`/`ViewPageContainer`/`AdminPage`/`FileVault`).
- [ ] One `ConfirmDialog` API in use across tasks/admin/rss/tabs; `useConfirmDialog`
      cancel path resolves.
- [ ] One `Badge` in use; `StatusBadge`/`JobStatusBadge` removed.
- [ ] No hand-rolled spinner markup remaining (SVG, border-div, or unicode).
- [ ] Single primary button color (brand) in use; `danger` variant exists.
- [ ] Dead code (`admin/ConfirmDialog`, `AdminTopBar`, `admin/StatusBadge`) removed.
- [ ] Governance check in place: no `fixed inset-0` outside `ui/`.
- [ ] `npm run test:run`, `format:check`, `build`, and `e2e/smoke.ts` green.
