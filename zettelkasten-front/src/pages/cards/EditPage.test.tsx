import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { BrowserRouter, MemoryRouter } from 'react-router-dom';
import { EditPage } from './EditPage';
import { UIStateProvider, useUIState } from '../../contexts/UIStateContext';
import { TagProvider } from '../../contexts/TagContext';
import { mockEndpoint } from '../../tests/fetchMock';
import { SchemaDefinition } from '../../models/Schema';

// Templates are the only network call made on mount for a new card.
vi.mock('../../api/templates', () => ({
  getTemplates: vi.fn(() => Promise.resolve([])),
}));

// Card body + its heavy sub-deps are irrelevant to the rail layout; stub them
// so the test stays focused and fast. FileListItem pulls in react-pdf which
// needs DOMMatrix (absent in happy-dom).
vi.mock('../../components/cards/CardBodyTextArea', () => ({
  CardBodyTextArea: () => <textarea data-testid="card-body" />,
  CardBodyTextAreaHandle: {},
}));

vi.mock('../../components/files/FileListItem', () => ({
  FileListItem: () => <div data-testid="file-list-item">file</div>,
}));

function renderEditPage() {
  return render(
    <BrowserRouter>
      <TagProvider testing testTags={[]}>
        <UIStateProvider>
          <EditPage newCard={true} />
        </UIStateProvider>
      </TagProvider>
    </BrowserRouter>,
  );
}

describe('EditPage — closable rail', () => {
  beforeEach(() => {
    // The hook writes ?pane= to the URL and the rail persists its open/closed
    // state to localStorage; reset both between tests so each starts on the
    // Metadata tab with the rail open.
    window.history.replaceState({}, '', '/');
    localStorage.setItem('zettelgarden-right-pane-open', 'true');
  });

  it('renders the Metadata, Links, and Files rail tabs by default', () => {
    renderEditPage();
    expect(screen.getByText('Metadata')).toBeInTheDocument();
    expect(screen.getByText('Links')).toBeInTheDocument();
    expect(screen.getByText('Files')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Close info pane' }),
    ).toBeInTheDocument();
  });

  it('hides the rail and reclaims full width when the pane is closed', () => {
    function Harness() {
      const { rightPaneOpen } = useUIState();
      return (
        <>
          <EditPage newCard={true} />
          <div data-testid="rail-state">
            {rightPaneOpen ? 'open' : 'closed'}
          </div>
        </>
      );
    }
    render(
      <BrowserRouter>
        <TagProvider testing testTags={[]}>
          <UIStateProvider>
            <Harness />
          </UIStateProvider>
        </TagProvider>
      </BrowserRouter>,
    );

    expect(screen.getByTestId('rail-state').textContent).toBe('open');
    expect(screen.getByText('Metadata')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Close info pane' }));

    expect(screen.getByTestId('rail-state').textContent).toBe('closed');
    // The rail tab strip + close affordance disappear with the panel.
    expect(screen.queryByText('Metadata')).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: 'Close info pane' }),
    ).not.toBeInTheDocument();
  });

  it('shows Source/Link in the rail (Metadata tab), not the main column', async () => {
    renderEditPage();
    // Default tab is Metadata; the Source input lives in the rail now. Wait
    // for the hook's mount effect to settle the smart default.
    const sourceInput = await screen.findByPlaceholderText('Source');
    expect(sourceInput).toBeInTheDocument();
    // The main column no longer has the old standalone Link section label.
    await waitFor(() => {
      expect(screen.getAllByText('Link').length).toBe(1);
    });
  });

  it('switches to the Links tab and shows References instead of metadata', async () => {
    renderEditPage();
    // Start on Metadata (smart default) — wait for it to settle.
    await screen.findByText('Card ID');

    fireEvent.click(screen.getByText('Links'));

    // Links tab shows the References backlink input.
    expect(await screen.findByText('References')).toBeInTheDocument();
    // Metadata content is hidden.
    expect(screen.queryByText('Card ID')).not.toBeInTheDocument();
    expect(screen.queryByText('Tags')).not.toBeInTheDocument();
  });
});

describe('EditPage new card with ?schema=', () => {
  const schema: SchemaDefinition = {
    id: 7,
    name: 'Book Review',
    slug: 'book-review',
    owner_id: 1,
    fields: [
      { name: 'Author', type: 'text', required: true },
      { name: 'Rating', type: 'number', required: false },
    ],
    created_at: new Date(),
    updated_at: new Date(),
    is_deleted: false,
  };

  function renderEditPageWithEntry(initialEntry: string) {
    return render(
      <MemoryRouter initialEntries={[initialEntry]}>
        <TagProvider testing testTags={[]}>
          <UIStateProvider>
            <EditPage newCard={true} />
          </UIStateProvider>
        </TagProvider>
      </MemoryRouter>,
    );
  }

  beforeEach(() => {
    // The schema table's "Add Card" navigates to card/new?schema=<id>; the
    // schemas list is what CardSchemaSection fetches to resolve the id.
    mockEndpoint('/schemas', [schema]);
  });

  it('pre-selects the schema and renders its structured fields', async () => {
    renderEditPageWithEntry('/app/card/new?schema=7');

    const select = await screen.findByLabelText('Schema');
    await waitFor(() => expect((select as HTMLSelectElement).value).toBe('7'));

    // The pre-attached schema's fields show up in the structured data editor
    // (required text fields append "(required)" to the placeholder).
    expect(await screen.findByPlaceholderText(/Author/)).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Rating')).toBeInTheDocument();
  });

  it('leaves the schema unset when no schema param is present', async () => {
    renderEditPageWithEntry('/app/card/new');

    const select = await screen.findByLabelText('Schema');
    expect((select as HTMLSelectElement).value).toBe('');
  });

  it('ignores a malformed schema param instead of attaching a wrong schema', async () => {
    renderEditPageWithEntry('/app/card/new?schema=7abc');

    const select = await screen.findByLabelText('Schema');
    await waitFor(() => expect((select as HTMLSelectElement).value).toBe(''));
  });

  it('ignores a negative schema param', async () => {
    renderEditPageWithEntry('/app/card/new?schema=-1');

    const select = await screen.findByLabelText('Schema');
    await waitFor(() => expect((select as HTMLSelectElement).value).toBe(''));
  });
});
