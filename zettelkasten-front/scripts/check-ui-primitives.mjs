#!/usr/bin/env node
/**
 * Governance check (Zettelgarden-z11.16): new dialogs/menus/buttons MUST
 * route through the primitives in src/components/ui/. This lightweight check
 * catches raw modal shells and raw Headless Dialog usage outside ui/.
 *
 * What it flags:
 *  1. Raw centered overlay shells: a `fixed inset-0` element with
 *     `items-center` + `justify-center` (the hand-rolled modal-centering
 *     pattern that ui/Modal replaces). This is the discriminator that
 *     separates real modal shells from bottom sheets / mobile drawers /
 *     sidebar backdrops / hover popovers / drag-and-drop overlays, which
 *     legitimately use `fixed inset-0` without centering.
 *  2. Raw Headless UI `Dialog` imports (components should use ui/Modal,
 *     which wraps Dialog). `Menu`/`Popover`/`Combobox`/`Transition` imports
 *     are allowed (popovers/autocompletes belong to Zettelgarden-12g).
 *
 * Allowlist: FileVault fullscreen/lightbox/upload overlays are intentional
 * non-dialog centered overlays (image preview, upload progress). Remove an
 * entry when that surface migrates to a primitive.
 *
 * Usage: node scripts/check-ui-primitives.mjs   (exit 1 on violations)
 */
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('..', import.meta.url));
const srcDir = join(root, 'src');

/** Intentional non-dialog centered overlays: { file, match, reason }. */
const ALLOWLIST = [
  {
    file: 'components/files/FileRender.tsx',
    match: 'fixed inset-0 flex items-center justify-center bg-black/50',
    reason: 'fullscreen text/markdown render overlay (FileVault)',
  },
  {
    file: 'components/files/FileListItem.tsx',
    match:
      'fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center',
    reason: 'image lightbox click-to-close (FileVault)',
  },
  {
    file: 'components/files/FileUpload.tsx',
    match:
      'fixed inset-0 bg-black bg-opacity-30 z-50 flex items-center justify-center',
    reason: 'upload progress overlay (FileVault)',
  },
  {
    file: 'components/files/FilePreview.tsx',
    match:
      'fixed inset-0 z-50 bg-black bg-opacity-75 flex items-center justify-center',
    reason: 'fullscreen preview overlay (FileVault)',
  },
  {
    file: 'pages/FileVault.tsx',
    match:
      'fixed inset-0 bg-black bg-opacity-30 z-50 flex items-center justify-center',
    reason: 'upload progress overlay (FileVault)',
  },
];

function walk(dir, out = []) {
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry);
    if (entry === 'node_modules' || entry.startsWith('.')) continue;
    if (statSync(full).isDirectory()) walk(full, out);
    else if (entry.endsWith('.tsx') || entry.endsWith('.ts')) out.push(full);
  }
  return out;
}

function isAllowed(file, line) {
  // file is relative to src/
  return ALLOWLIST.some((a) => file === a.file && line.includes(a.match));
}

const violations = [];
for (const file of walk(srcDir)) {
  const rel = relative(srcDir, file);
  if (rel.startsWith('ui/')) continue;
  const lines = readFileSync(file, 'utf8').split('\n');
  lines.forEach((line, i) => {
    const centeredShell =
      /fixed inset-0/.test(line) &&
      /items-center/.test(line) &&
      /justify-center/.test(line);
    const rawHeadlessDialog = /from '@headlessui\/react'[^;]*\bDialog\b/.test(
      line,
    );
    if ((centeredShell || rawHeadlessDialog) && !isAllowed(rel, line.trim())) {
      violations.push(`${rel}:${i + 1}: ${line.trim()}`);
    }
  });
}

if (violations.length > 0) {
  console.error(
    'UI primitives governance check FAILED — raw modal shells / Headless Dialog outside src/components/ui/:\n',
  );
  violations.forEach((v) => console.error('  ' + v));
  console.error(
    '\nRoute new dialogs through ui/Modal (see docs/ui-primitives-how-to-use.md).',
  );
  process.exit(1);
}
console.log('UI primitives governance check passed.');
