# UI primitives — how to use

> Decision recorded for [Zettelgarden-z11.4]: the **single primary-button color
> is `bg-palette-dark`** (the brand teal `#38a3a5`, hover `bg-palette-darkest`).
> Ad-hoc `bg-blue-*/bg-indigo-*` primary buttons are being swept out as
> surfaces migrate. `bg-palette-dark` is already the `primary` variant of
> `ui/Button`.

## The rule

**New dialogs, menus, and buttons MUST route through the primitives in
`src/components/ui/`.** Do not hand-write raw `fixed inset-0` modal shells,
raw styled `<button>` elements, custom menu markup, spinner markup, status
pills, or label+input+error markup. If the primitive doesn't cover your case,
extend the primitive (or file an issue), don't fork the markup.

This is enforced by code review today and will be backed by a CI grep check
(no `fixed inset-0` outside `ui/`; see Zettelgarden-z11.16) once Phase D
lands. ESLint rules may follow later.

## The primitives

| Primitive                                          | Use for                        | Notes                                                                                                                                                          |
| -------------------------------------------------- | ------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ui/Button`                                        | Any interactive button         | Variants: `primary` (brand `bg-palette-dark`), `secondary`, `outline`, `danger`. Sizes: `small`/`medium`/`large`; the `min-h-[44px]` touch target is built in. |
| `ui/Modal`                                         | Any dialog shell               | Overlay, sizing (`size` = `sm`…`4xl`), Escape, focus trap, scroll-lock, aria. Controlled via `open`/`onClose`.                                                 |
| `ui/ConfirmDialog`                                 | Any yes/no confirmation        | Built on Modal. `variant` = `danger`/`warning`/`info`, optional `requireCheckbox`, `details`. Also `useConfirmDialog()` for `await confirm(...)` → `boolean`.  |
| `ui/Menu`                                          | Dropdown menus / kebab actions | Trigger button + panel; `MenuItem` has the 44px touch target and active highlight; keyboard nav built in.                                                      |
| `ui/Dropdown`                                      | Select-style pickers           | Shows current value + chevron, options with check mark on the selected one.                                                                                    |
| `ui/Badge`                                         | Status pills                   | `color` = `success`/`warning`/`error`/`info`/`neutral`, optional `dot` / `pulse`.                                                                              |
| `ui/Spinner`                                       | Any loading indicator          | `size` = `sm`/`md`/`lg`/`xl`; color via `className="text-…"`; `role="status"` + sr-only label.                                                                 |
| `ui/Field` / `ui/Input` / `ui/Select` / `ui/Label` | Forms                          | Field wires label + control + error/help slot; `hasError` on Input/Select shows the red state.                                                                 |

## How to use

```tsx
import { Button } from '../components/ui/Button';
import { Modal } from '../components/ui/Modal';
import { ConfirmDialog } from '../components/ui/ConfirmDialog';

// Buttons
<Button variant="primary" onClick={save}>Save</Button>
<Button variant="danger" onClick={remove}>Delete</Button>

// Dialog shell (open/close controlled)
<Modal open={isOpen} onClose={close} size="md">
  …content…
</Modal>

// Confirmation
<ConfirmDialog
  isOpen={showConfirm}
  onClose={() => setShowConfirm(false)}
  onConfirm={doIt}
  title="Delete task"
  message="This cannot be undone."
  variant="danger"
  confirmText="Delete"
/>
```

### ConfirmDialog with a promise

```tsx
const { confirm, Dialog } = useConfirmDialog();

async function handleDelete() {
  const ok = await confirm({
    title: 'Delete task',
    message: 'This cannot be undone.',
    variant: 'danger',
    confirmText: 'Delete',
  });
  if (ok) { …actually delete… }
}

// render <Dialog /> once near the top level
```

`confirm(...)` resolves `true` on confirm and `false` on cancel/Escape — it
never hangs.

## Migration status

- Phase A (primitives + this doc): done (Zettelgarden-z11.1…z11.8).
- Phase B (seed adoption on highest-traffic surfaces): Zettelgarden-z11.9, z11.10.
- Phase C (sweep dialogs → Modal, menus → Menu, pills → Badge, spinners →
  Spinner, forms → Field, delete dead code): Zettelgarden-z11.11…z11.14.
- Phase D (governance check, palette normalization): Zettelgarden-z11.16, z11.12.

See `docs/plans/2026-08-09-frontend-component-consistency.md` for the full plan.
