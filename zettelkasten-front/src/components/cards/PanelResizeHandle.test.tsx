import { describe, it, expect } from 'vitest';
import { render, fireEvent } from '@testing-library/react';
import { useState } from 'react';
import { PanelResizeHandle } from './PanelResizeHandle';

/**
 * Harness that mimics how ViewPage + UIStateContext wire the handle up:
 * `onResize` is an inline function, so it gets a NEW identity on every
 * render — exactly what previously tore the document listeners off
 * mid-drag and limited resizing to the first few pixels.
 */
function Harness() {
  const [width, setWidth] = useState(400);
  return (
    <>
      <div data-testid="width">{width}</div>
      <PanelResizeHandle width={width} onResize={(w) => setWidth(w)} />
    </>
  );
}

describe('PanelResizeHandle', () => {
  it('continues resizing across multiple mousemove events that each re-render', () => {
    const { getByRole, getByTestId } = render(<Harness />);
    const handle = getByRole('separator');

    fireEvent.mouseDown(handle, { clientX: 500 });
    // Drag left 50px: panel (to the right of the handle) widens to 450.
    fireEvent.mouseMove(document, { clientX: 450 });
    expect(getByTestId('width').textContent).toBe('450');

    // Keep dragging left past the re-render: must keep widening.
    fireEvent.mouseMove(document, { clientX: 400 });
    expect(getByTestId('width').textContent).toBe('500');

    fireEvent.mouseMove(document, { clientX: 300 });
    expect(getByTestId('width').textContent).toBe('600');

    fireEvent.mouseUp(document);
  });

  it('clamps the width to the min/max while dragging', () => {
    const { getByRole, getByTestId } = render(<Harness />);
    const handle = getByRole('separator');

    fireEvent.mouseDown(handle, { clientX: 500 });
    // Drag far right (narrowing): clamps to min.
    fireEvent.mouseMove(document, { clientX: 1000 });
    expect(getByTestId('width').textContent).toBe('280');

    // Drag far left (widening): clamps to max.
    fireEvent.mouseMove(document, { clientX: -500 });
    expect(getByTestId('width').textContent).toBe('640');

    fireEvent.mouseUp(document);
  });

  it('stops resizing after mouseup', () => {
    const { getByRole, getByTestId } = render(<Harness />);
    const handle = getByRole('separator');

    fireEvent.mouseDown(handle, { clientX: 500 });
    fireEvent.mouseMove(document, { clientX: 400 });
    expect(getByTestId('width').textContent).toBe('500');

    fireEvent.mouseUp(document);
    fireEvent.mouseMove(document, { clientX: 300 });
    // Width unchanged after the drag ends.
    expect(getByTestId('width').textContent).toBe('500');
  });

  it('resets to the midpoint on double click', () => {
    const { getByRole, getByTestId } = render(<Harness />);
    const handle = getByRole('separator');

    fireEvent.doubleClick(handle);
    expect(getByTestId('width').textContent).toBe('460');
  });
});
