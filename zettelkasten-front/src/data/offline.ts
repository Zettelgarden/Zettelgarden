/**
 * Online-only reads in the desktop app degrade gracefully when the server is
 * unreachable (Phase 2b — offline-writable scope is cards/tasks/tags; the
 * rest — entities, files, references, summaries — is available online).
 *
 * The web app never swallows errors (thin client, unchanged behavior); in
 * the desktop shell a network failure (or navigator offline) falls back to
 * an empty/neutral value so card view/edit flows never dead-end on a
 * satellite fetch while the local mirror serves the card itself.
 */

import { isNetworkError } from '../api/errors';
import { isNativeShell } from './tauriStorageAdapter';

export async function graceful<T>(
  fallback: T,
  fn: () => Promise<T>,
): Promise<T> {
  if (!isNativeShell()) return fn();
  try {
    return await fn();
  } catch (err) {
    if (isNetworkError(err) || navigator.onLine === false) {
      return fallback;
    }
    throw err;
  }
}

/** Empty categorized references for offline card views. */
export const EMPTY_REFERENCES = {
  bidirectional: [],
  outgoing: [],
  incoming: [],
};
