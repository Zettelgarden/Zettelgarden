# Frontend Testing Guide

This guide outlines the testing strategy and best practices for the Zettelgarden frontend application.

## Testing Stack

- **Test Runner**: Vitest
- **Testing Library**: React Testing Library
- **Environment**: Happy DOM
- **Coverage**: V8
- **User Interactions**: @testing-library/user-event

## Running Tests

```bash
# Run tests once
npm run test:run

# Run tests in watch mode
npm run test:watch

# Run tests with coverage
npm run test:coverage

# Run tests with UI (requires @vitest/ui)
npm run test:ui
```

## Testing Strategy

### 1. Unit Tests

Test individual components, utilities, and functions in isolation.

**Examples:**

- Utils functions (dates, strings, cards, tasks)
- Individual components (buttons, inputs, menus)
- API functions
- Context providers

### 2. Integration Tests

Test how components work together and interact with contexts.

**Examples:**

- Components with their contexts
- Page-level components
- Complex user flows

### 3. What to Test

#### ✅ DO Test:

- **User interactions**: clicks, typing, form submissions
- **Conditional rendering**: showing/hiding elements based on state
- **Props handling**: component behavior with different props
- **State changes**: component state updates
- **Context integration**: components using React contexts
- **API integration**: mocked API calls and responses
- **Error states**: error boundaries and error handling
- **Accessibility**: basic a11y attributes and behavior

#### ❌ DON'T Test:

- **Implementation details**: internal state, private methods
- **Third-party libraries**: React Router, headlessui components
- **Styling**: CSS classes, visual appearance
- **Browser APIs**: unless specifically testing integration

### 4. Testing Patterns

#### Component Testing Pattern

```tsx
import React from 'react';
import { describe, it, expect, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '../../tests/utils';
import { MyComponent } from './MyComponent';

describe('MyComponent', () => {
  const user = userEvent.setup();

  it('renders correctly', () => {
    renderWithProviders(<MyComponent />);
    expect(screen.getByText('Expected Text')).toBeInTheDocument();
  });

  it('handles user interaction', async () => {
    const mockFn = vi.fn();
    renderWithProviders(<MyComponent onAction={mockFn} />);

    await user.click(screen.getByRole('button'));
    expect(mockFn).toHaveBeenCalled();
  });
});
```

#### API Testing Pattern

```tsx
// Mock the API module
vi.mock('../../api/cards', () => ({
  getCard: vi.fn(() => Promise.resolve(mockCard)),
  saveCard: vi.fn(() => Promise.resolve({ id: 1 })),
}));

it('handles API calls', async () => {
  const { getCard } = await import('../../api/cards');

  renderWithProviders(<CardComponent cardId="123" />);

  await waitFor(() => {
    expect(getCard).toHaveBeenCalledWith('123');
  });
});
```

#### Context Testing Pattern

```tsx
const wrapper = ({ children }) => (
  <TagProvider testing={true} testTags={mockTags}>
    {children}
  </TagProvider>
);

it('uses context correctly', () => {
  const { result } = renderHook(() => useTagContext(), { wrapper });
  expect(result.current.tags).toEqual(mockTags);
});
```

### 5. File Structure

```
src/
├── components/
│   ├── Button/
│   │   ├── Button.tsx
│   │   └── Button.test.tsx
│   └── cards/
│       ├── CardItem.tsx
│       └── __tests__/
│           └── CardItem.test.tsx
├── pages/
│   └── SearchPage/
│       ├── SearchPage.tsx
│       └── SearchPage.test.tsx
├── contexts/
│   └── __tests__/
│       └── TagContext.test.tsx
├── utils/
│   ├── dates.ts
│   └── dates.test.ts
└── tests/
    ├── setup.ts
    ├── utils.tsx
    └── data.ts
```

### 6. Common Testing Utilities

#### renderWithProviders

Use this for components that need React contexts:

```tsx
import { renderWithProviders } from '../../tests/utils';

renderWithProviders(<MyComponent />);
```

#### Mocking APIs

```tsx
vi.mock('../../api/cards', () => ({
  getCard: vi.fn(),
  saveCard: vi.fn(),
}));
```

#### Testing User Events

```tsx
const user = userEvent.setup();

// Click
await user.click(screen.getByRole('button'));

// Type
await user.type(screen.getByRole('textbox'), 'hello world');

// Select
await user.selectOptions(screen.getByRole('combobox'), 'option1');
```

### 7. Coverage Goals

- **Overall coverage**: 80%+
- **Critical paths**: 90%+
- **Utilities**: 95%+
- **New features**: 100%

### 8. Best Practices

1. **Write tests before or alongside code** (TDD/BDD)
2. **Test behavior, not implementation**
3. **Use descriptive test names** that explain what the test does
4. **Keep tests simple and focused** - one concept per test
5. **Use real user interactions** instead of calling props directly
6. **Clean up after tests** - clear mocks, reset state
7. **Use data attributes** for test targeting when needed: `data-testid`

### 9. Common Gotchas

1. **Async operations**: Always await user events and API calls
2. **Context providers**: Use renderWithProviders for components using contexts
3. **Router dependencies**: Wrap components needing routing in MemoryRouter
4. **Timers**: Mock timers when testing time-dependent code
5. **External dependencies**: Mock all external API calls

### 10. Testing Checklist for New Features

- [ ] Unit tests for new utilities/functions
- [ ] Component tests for new UI components
- [ ] Integration tests for complex features
- [ ] API mocking for external dependencies
- [ ] Error state testing
- [ ] User interaction testing
- [ ] Accessibility testing (basic)
- [ ] Coverage meets minimum requirements

### 11. Example Test Scenarios

#### Form Testing

```tsx
it('submits form with correct data', async () => {
  const mockSubmit = vi.fn();
  renderWithProviders(<MyForm onSubmit={mockSubmit} />);

  await user.type(screen.getByLabelText('Title'), 'Test Title');
  await user.click(screen.getByRole('button', { name: /submit/i }));

  expect(mockSubmit).toHaveBeenCalledWith({
    title: 'Test Title',
  });
});
```

#### Loading State Testing

```tsx
it('shows loading state', async () => {
  renderWithProviders(<AsyncComponent />);

  expect(screen.getByText('Loading')).toBeInTheDocument();

  await waitFor(() => {
    expect(screen.queryByText('Loading')).not.toBeInTheDocument();
  });
});
```

#### Error State Testing

```tsx
it('handles errors gracefully', async () => {
  const mockAPI = vi.fn().mockRejectedValue(new Error('API Error'));

  renderWithProviders(<ComponentWithAPI apiCall={mockAPI} />);

  await waitFor(() => {
    expect(screen.getByText(/error/i)).toBeInTheDocument();
  });
});
```

## Getting Started

1. **Run existing tests**: `npm run test:run`
2. **Check coverage**: `npm run test:coverage`
3. **Start with utils**: Write tests for utility functions first
4. **Add component tests**: Test your most critical components
5. **Gradually increase coverage**: Aim for 80%+ overall coverage

## Resources

- [Vitest Documentation](https://vitest.dev/)
- [React Testing Library](https://testing-library.com/docs/react-testing-library/intro/)
- [Testing Best Practices](https://kentcdodds.com/blog/common-mistakes-with-react-testing-library)
