import React, { ReactElement } from 'react';
import { render, act } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { UIStateProvider } from '../contexts/UIStateContext';
import { DialogStateProvider } from '../contexts/DialogStateContext';
import { TaskProvider } from '../contexts/TaskContext';
import { TagProvider } from '../contexts/TagContext';
import { StatusProvider } from '../contexts/StatusContext';
import { AuthProvider } from '../contexts/AuthContext';
import { RSSProvider } from '../contexts/RSSContext';
import { ToastProvider } from '../components/toast/ToastContext';
import { sampleTasks, sampleTags } from '../tests/data';

function AllTheProviders({ children }) {
  return (
    <BrowserRouter>
      <AuthProvider>
        <TagProvider testing={true} testTags={sampleTags()}>
          <TaskProvider testing={true} testTasks={sampleTasks()}>
            <StatusProvider>
              <RSSProvider>
                <ToastProvider>
                  <UIStateProvider>
                    <DialogStateProvider>{children}</DialogStateProvider>
                  </UIStateProvider>
                </ToastProvider>
              </RSSProvider>
            </StatusProvider>
          </TaskProvider>
        </TagProvider>
      </AuthProvider>
    </BrowserRouter>
  );
}

const customRender = (ui: ReactElement, options = {}) =>
  render(ui, { wrapper: AllTheProviders, ...options });

/**
 * Flush pending async work (provider mount-fetches, etc.) inside act() so
 * their state updates don't emit "not wrapped in act(...)" warnings.
 * Call `await settle()` after render() in tests that mount the provider
 * stack and don't otherwise wait for fetched data.
 */
export async function settle() {
  await act(async () => {
    // Yield through a macrotask so multi-hop promise chains (fetch ->
    // response.json/text -> setState) complete inside the act scope.
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
}

// Re-export everything from @testing-library/react
export * from '@testing-library/react';

// Override render method
export { customRender as renderWithProviders };
