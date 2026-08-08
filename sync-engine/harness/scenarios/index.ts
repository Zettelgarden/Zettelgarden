import type { Scenario } from './context';
import { offlineGapScenario } from './01-offline-gap';
import { concurrentLwwScenario } from './02-concurrent-lww';
import { linkedRowsScenario } from './03-linked-rows';
import { cardIdRenameScenario } from './04-card-id-rename';
import { selfEchoScenario } from './05-self-echo';
import { tagRenameScenario } from './06-tag-rename';
import { offlineDeleteScenario } from './07-offline-delete';

/** All Phase 1b convergence scenarios, run in order against the live backend. */
export const scenarios: Scenario[] = [
  offlineGapScenario,
  concurrentLwwScenario,
  linkedRowsScenario,
  cardIdRenameScenario,
  selfEchoScenario,
  tagRenameScenario,
  offlineDeleteScenario,
];
