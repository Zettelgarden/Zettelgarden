/**
 * React Query Provider component
 *
 * Wraps the application with QueryClientProvider and optionally
 * ReactQueryDevtools for development.
 */

import React from 'react';
import { QueryClientProvider as TanStackQueryClientProvider } from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { createQueryClient } from '../api/queryClient';

interface QueryClientProviderProps {
  children: React.ReactNode;
  /**
   * Enable ReactQueryDevtools
   * @default process.env.NODE_ENV === 'development'
   */
  enableDevtools?: boolean;
}

/**
 * Create a singleton query client instance
 * In production, you might want to create a new client per request for SSR
 */
const queryClient = createQueryClient();

/**
 * QueryClientProvider component
 *
 * Place this component high in your component tree, wrapping all components
 * that need to use React Query hooks.
 *
 * @example
 * ```tsx
 * // In src/index.tsx or src/main.tsx
 * import { QueryClientProvider } from './components/ReactQueryDevtools';
 *
 * ReactDOM.createRoot(document.getElementById('root')!).render(
 *   <React.StrictMode>
 *     <BrowserRouter>
 *       <AuthProvider>
 *         <QueryClientProvider>
 *           <App />
 *         </QueryClientProvider>
 *       </AuthProvider>
 *     </BrowserRouter>
 *   </React.StrictMode>
 * );
 * ```
 */
export function QueryClientProvider({
  children,
  enableDevtools = import.meta.env.DEV,
}: QueryClientProviderProps) {
  return (
    <TanStackQueryClientProvider client={queryClient}>
      {children}
      {enableDevtools && (
        <ReactQueryDevtools
          initialIsOpen={false}
          position="bottom-right"
        />
      )}
    </TanStackQueryClientProvider>
  );
}
