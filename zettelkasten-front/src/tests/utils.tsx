import React, { ReactElement } from 'react';
import { render } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { UIStateProvider } from '../contexts/UIStateContext';
import { DialogStateProvider } from '../contexts/DialogStateContext';
import { TaskProvider } from '../contexts/TaskContext';
import { TagProvider } from '../contexts/TagContext';
import { StatusProvider } from '../contexts/StatusContext';
import { AuthProvider } from '../contexts/AuthContext';
import { RSSProvider } from '../contexts/RSSContext';
import { sampleTasks, sampleTags } from '../tests/data';

function AllTheProviders({ children }) {
  return (
    <BrowserRouter>
      <AuthProvider>
        <TagProvider testing={true} testTags={sampleTags()}>
          <TaskProvider testing={true} testTasks={sampleTasks()}>
            <StatusProvider>
              <RSSProvider>
                <UIStateProvider>
                  <DialogStateProvider>{children}</DialogStateProvider>
                </UIStateProvider>
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

// Re-export everything from @testing-library/react
export * from '@testing-library/react';

// Override render method
export { customRender as renderWithProviders };
