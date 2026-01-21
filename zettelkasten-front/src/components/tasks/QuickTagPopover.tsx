import React from "react";
import { createPortal } from "react-dom";
import { useTagContext } from "../../contexts/TagContext";

export type QuickTagTrigger = {
  // index of the triggering '#'
  start: number;
  // cursor position (end of the token being typed)
  end: number;
  // text after '#', up to cursor (may be empty)
  query: string;
};

export function getQuickTagTrigger(
  title: string,
  cursor: number,
): QuickTagTrigger | null {
  const safeCursor = Math.max(0, Math.min(cursor, title.length));
  const prefix = title.slice(0, safeCursor);
  const hashIndex = prefix.lastIndexOf("#");
  if (hashIndex === -1) return null;

  // Word-start trigger: start of string or preceded by whitespace
  if (hashIndex > 0 && !/\s/.test(title[hashIndex - 1])) return null;

  // Treat multiple consecutive hashes as literal text
  if (hashIndex > 0 && title[hashIndex - 1] === "#") return null;

  // Only while cursor is still within the same token (no whitespace after '#')
  if (/\s/.test(prefix.slice(hashIndex))) return null;

  const query = title.slice(hashIndex + 1, safeCursor);
  if (query.includes("#")) return null;

  return { start: hashIndex, end: safeCursor, query };
}

function escapeRegExp(input: string) {
  return input.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

export function applyQuickTagSelection(params: {
  title: string;
  trigger: QuickTagTrigger;
  selectedTagName: string;
}): { nextTitle: string; nextCursor: number; didInsert: boolean } {
  const { title, trigger } = params;
  const clean = params.selectedTagName.replace(/^#/, "").trim();
  if (!clean) {
    return { nextTitle: title, nextCursor: trigger.end, didInsert: false };
  }

  const tokenRegex = new RegExp(`(^|\\s)#${escapeRegExp(clean)}(?=\\s|$)`);
  if (tokenRegex.test(title)) {
    return { nextTitle: title, nextCursor: trigger.end, didInsert: false };
  }

  const before = title.slice(0, trigger.start);
  const after = title.slice(trigger.end);

  // Defensive spacing: ensure a leading space before '#'
  const needsLeadingSpace = before.length > 0 && !/\s$/.test(before);

  const tagToken = `#${clean}`;
  let nextTitle = `${before}${needsLeadingSpace ? " " : ""}${tagToken} ${after}`;

  // Normalize runs of whitespace but preserve the single trailing space after insertion.
  nextTitle = nextTitle.replace(/\s{2,}/g, " ").replace(/^\s+/, "");

  const insertedIndex = nextTitle.indexOf(tagToken, Math.max(0, trigger.start - 1));
  const nextCursor =
    insertedIndex === -1
      ? nextTitle.length
      : Math.min(nextTitle.length, insertedIndex + tagToken.length + 1);

  return { nextTitle, nextCursor, didInsert: true };
}

export function filterAndSortTagNames(
  tags: Array<{ name: string }>,
  query: string,
): string[] {
  const q = query.toLowerCase();
  const normalized = tags
    .map((t) => t.name.replace(/^#/, ""))
    .filter((name) => name.length > 0);

  const filtered = normalized.filter((name) =>
    name.toLowerCase().includes(q),
  );

  filtered.sort((a, b) => {
    const aLower = a.toLowerCase();
    const bLower = b.toLowerCase();
    const aPrefix = q.length > 0 && aLower.startsWith(q);
    const bPrefix = q.length > 0 && bLower.startsWith(q);

    if (aPrefix !== bPrefix) return aPrefix ? -1 : 1;
    return aLower.localeCompare(bLower);
  });

  return filtered;
}

function getCaretViewportRect(params: {
  input: HTMLInputElement;
  value: string;
  cursor: number;
}): { left: number; top: number; height: number; inputRect: DOMRect } {
  const { input, value } = params;
  const cursor = Math.max(0, Math.min(params.cursor, value.length));

  const inputRect = input.getBoundingClientRect();
  const style = window.getComputedStyle(input);

  const mirror = document.createElement("div");
  mirror.style.position = "absolute";
  mirror.style.visibility = "hidden";
  mirror.style.whiteSpace = "pre";
  mirror.style.top = "0";
  mirror.style.left = "0";

  // Copy relevant text/layout styles
  mirror.style.fontFamily = style.fontFamily;
  mirror.style.fontSize = style.fontSize;
  mirror.style.fontWeight = style.fontWeight;
  mirror.style.fontStyle = style.fontStyle;
  mirror.style.letterSpacing = style.letterSpacing;
  mirror.style.textTransform = style.textTransform;
  mirror.style.padding = style.padding;
  mirror.style.border = style.border;
  mirror.style.boxSizing = style.boxSizing;
  mirror.style.width = style.width;

  // Mirror scrolling behavior
  mirror.style.overflow = "hidden";

  const beforeText = value.slice(0, cursor);
  const afterText = value.slice(cursor) || ".";

  mirror.textContent = beforeText;
  const marker = document.createElement("span");
  marker.textContent = afterText[0];
  mirror.appendChild(marker);

  document.body.appendChild(mirror);

  // Account for horizontal scrolling in the input
  const caretLeftInInput = marker.offsetLeft - input.scrollLeft;
  const caretTopInInput = marker.offsetTop - input.scrollTop;

  document.body.removeChild(mirror);

  return {
    left: inputRect.left + caretLeftInInput,
    top: inputRect.top + caretTopInInput,
    height: inputRect.height,
    inputRect,
  };
}

export interface QuickTagPopoverProps {
  open: boolean;
  anchorInputRef: React.RefObject<HTMLInputElement>;
  titleValue: string;
  cursorPosition: number;
  trigger: QuickTagTrigger | null;
  onSelectTag: (selectedTagName: string, cursorPosition: number) => void;
  onRequestClose: () => void;
}

export function QuickTagPopover({
  open,
  anchorInputRef,
  titleValue,
  cursorPosition,
  trigger,
  onSelectTag,
  onRequestClose,
}: QuickTagPopoverProps) {
  const { tags } = useTagContext();
  const popoverRef = React.useRef<HTMLDivElement>(null);

  const query = trigger?.query ?? "";
  const suggestions = React.useMemo(
    () => filterAndSortTagNames(tags, query),
    [tags, query],
  );

  const [activeIndex, setActiveIndex] = React.useState(0);

  React.useEffect(() => {
    if (!open) return;
    setActiveIndex(0);
  }, [open, query, suggestions.length]);

  const [pos, setPos] = React.useState<{ left: number; top: number } | null>(
    null,
  );

  React.useLayoutEffect(() => {
    if (!open || !trigger) return;
    const input = anchorInputRef.current;
    const pop = popoverRef.current;
    if (!input || !pop) return;

    const caret = getCaretViewportRect({
      input,
      value: titleValue,
      cursor: cursorPosition,
    });

    const popRect = pop.getBoundingClientRect();

    const margin = 8;
    let left = caret.left;
    left = Math.max(margin, Math.min(left, window.innerWidth - popRect.width - margin));

    const belowTop = caret.top + caret.height + 6;
    let top = belowTop;

    if (belowTop + popRect.height > window.innerHeight - margin) {
      top = Math.max(margin, caret.inputRect.top - popRect.height - 6);
    }

    setPos({ left, top });
  }, [open, trigger, anchorInputRef, titleValue, cursorPosition, suggestions.length]);

  // Close on outside click
  React.useEffect(() => {
    if (!open) return;

    function onMouseDown(e: MouseEvent) {
      const input = anchorInputRef.current;
      const pop = popoverRef.current;
      const target = e.target as Node | null;

      if (!target) return;
      if (input && input.contains(target)) return;
      if (pop && pop.contains(target)) return;

      onRequestClose();
    }

    document.addEventListener("mousedown", onMouseDown);
    return () => document.removeEventListener("mousedown", onMouseDown);
  }, [open, anchorInputRef, onRequestClose]);

  // Keyboard navigation on the title input
  React.useEffect(() => {
    if (!open) return;
    const input = anchorInputRef.current;
    if (!input) return;

    function onKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") {
        e.preventDefault();
        onRequestClose();
        return;
      }

      if (suggestions.length === 0) return;

      if (e.key === "ArrowDown") {
        e.preventDefault();
        setActiveIndex((i) => Math.min(i + 1, suggestions.length - 1));
        return;
      }

      if (e.key === "ArrowUp") {
        e.preventDefault();
        setActiveIndex((i) => Math.max(i - 1, 0));
        return;
      }

      if (e.key === "Enter") {
        e.preventDefault();

        const clampedIndex = Math.max(
          0,
          Math.min(activeIndex, Math.max(0, suggestions.length - 1)),
        );

        const selected = suggestions[clampedIndex];
        if (selected) {
          onSelectTag(selected, cursorPosition);
          onRequestClose();
        }
      }
    }

    input.addEventListener("keydown", onKeyDown);
    return () => input.removeEventListener("keydown", onKeyDown);
  }, [
    open,
    anchorInputRef,
    suggestions,
    activeIndex,
    cursorPosition,
    onSelectTag,
    onRequestClose,
  ]);

  if (!open || !trigger) return null;

  const content = (
    <div
      ref={popoverRef}
      className="rounded-lg shadow-lg border border-gray-200 bg-white w-64"
      style={{
        position: "fixed",
        left: pos?.left ?? -9999,
        top: pos?.top ?? -9999,
        // Must sit above create-task overlay (z-index: 1000) and HeadlessUI dialogs.
        zIndex: 2000,
      }}
    >
      <div className="max-h-48 overflow-y-auto py-1">
        {suggestions.length === 0 ? (
          <div className="px-3 py-2 text-sm text-gray-500">No tags found</div>
        ) : (
          suggestions.map((name, idx) => {
            const isActive = idx === activeIndex;
            return (
              <button
                key={name}
                type="button"
                className={`w-full text-left px-3 py-2 text-sm ${
                  isActive
                    ? "bg-purple-600 text-white"
                    : "text-gray-700 hover:bg-purple-50"
                }`}
                onMouseEnter={() => setActiveIndex(idx)}
                onMouseDown={(e) => {
                  // Prevent input blur
                  e.preventDefault();
                  onSelectTag(name, cursorPosition);
                  onRequestClose();
                }}
              >
                #{name}
              </button>
            );
          })
        )}
      </div>
    </div>
  );

  // Avoid clipping by rendering at body level.
  return typeof document !== "undefined" ? createPortal(content, document.body) : content;
}
