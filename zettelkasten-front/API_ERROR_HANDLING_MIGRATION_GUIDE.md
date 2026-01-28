# API Error Handling Migration Guide

This guide explains the new API error handling system and how to migrate from the old `checkStatus` pattern.

## Overview

The old API error handling in `common.ts` had a problematic side effect: it directly called `window.location.reload()` for 401/422 errors. This violated separation of concerns and made testing difficult.

The new system:
- **API Layer** (`src/api/client.ts`): Returns typed error objects instead of triggering side effects
- **UI Layer** (`src/components/api/ApiErrorHandler.tsx`): Handles errors appropriately (toasts, redirects, logout)

## Key Changes

### Old Pattern (Deprecated)
```typescript
import { checkStatus } from './common';

fetch(url, { headers })
  .then(checkStatus)  // Side effect: could reload page
  .then(response => response.json());
```

### New Pattern
```typescript
import { apiClient, getData } from './client';
import { useApiErrorHandler } from '../components/api/ApiErrorHandler';

const { handleApiError } = useApiErrorHandler();

try {
  const data = await getData(apiClient.get('/endpoint'));
  // Use data
} catch (error) {
  handleApiError(error);  // Toasts, redirects, logout handled here
}
```

## File Structure

```
src/api/
├── client.ts           # New API client (use this)
├── errors.ts           # Error type definitions
├── common.ts           # Deprecated (still works for backward compatibility)
└── *.ts                # API modules (users.ts, cards.ts, tasks.ts, etc.)

src/components/api/
└── ApiErrorHandler.tsx # React hook for error handling
```

## API Client Reference

### Basic Methods

```typescript
import { apiClient, getData } from './client';

// GET request
const { data, response } = await apiClient.get<User>('/users/1');
const userData = await getData(apiClient.get<User>('/users/1'));

// POST request
const newUser = await getData(apiClient.post<User>('/users', userData));

// PUT request
const updatedUser = await getData(apiClient.put<User>('/users/1', updates));

// DELETE request
await apiClient.delete('/users/1');

// PATCH request
const patched = await getData(apiClient.patch<User>('/users/1', partialUpdate));
```

### Request Options

```typescript
// Query parameters
const users = await getData(apiClient.get<UsersResponse>('/users', {
  params: { page: 1, limit: 10, search: 'test' }
}));

// Skip auth header (for public endpoints)
const data = await getData(apiClient.post('/login', credentials, {
  skipAuth: true
}));

// Custom headers
const result = await apiClient.get('/endpoint', {
  headers: { 'X-Custom-Header': 'value' }
});
```

### Response Handling

```typescript
// Full response with status code and headers
const { data, response } = await apiClient.get<User>('/users/1');

console.log(response.status);        // 200
console.log(response.headers);       // Headers object
console.log(data);                   // User object

// Just get the data (most common case)
const user = await getData(apiClient.get<User>('/users/1'));
```

## Error Types

All error types extend `APIError`:

| Error Type | Status Code | Use Case |
|------------|-------------|----------|
| `AuthError` | 401 | Authentication failed |
| `TokenValidationError` | 422 | Token invalid/expired |
| `NetworkError` | 0 | Connection failed |
| `ValidationError` | 400 | Invalid input |
| `NotFoundError` | 404 | Resource not found |
| `ServerError` | 500+ | Server error |

### Type Guards

```typescript
import { isAuthError, isNetworkError, isAPIError } from './errors';

try {
  await getData(apiClient.get('/endpoint'));
} catch (error) {
  if (isAuthError(error)) {
    console.log('Auth failed:', error.message);
  } else if (isNetworkError(error)) {
    console.log('Connection failed');
  } else if (isAPIError(error)) {
    console.log(`API error: ${error.status}`);
  }
}
```

## Error Handling in Components

### Using useApiErrorHandler Hook

```typescript
import { useApiErrorHandler } from '../components/api/ApiErrorHandler';
import { getData } from '../../api/client';

function MyComponent() {
  const { handleApiError } = useApiErrorHandler();

  const fetchData = async () => {
    try {
      const data = await getData(apiClient.get('/endpoint'));
      setState(data);
    } catch (error) {
      // Automatically shows toast, handles auth errors, redirects
      handleApiError(error);
    }
  };

  return <button onClick={fetchData}>Load Data</button>;
}
```

### Error Handling Options

```typescript
// Show toast (default)
handleApiError(error);

// No toast, just logout/redirect
handleApiError(error, { showToast: false });

// No redirect, just toast and logout state
handleApiError(error, { redirectOnError: false });

// Custom toast message
handleApiError(error, { customMessage: 'Could not load user data' });
```

## Migration Examples

### Example 1: Simple GET Request

**Before:**
```typescript
import { checkStatus } from './common';

const user = await fetch('/users/1', {
  headers: { Authorization: `Bearer ${token}` }
})
  .then(checkStatus)
  .then(r => r.json());
```

**After:**
```typescript
import { getData } from './client';

const user = await getData(apiClient.get('/users/1'));
```

### Example 2: POST Request with Error Handling

**Before:**
```typescript
import { checkStatus } from './common';

try {
  const newUser = await fetch('/users', {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(userData),
  })
    .then(checkStatus)
    .then(r => r.json());
} catch (error) {
  // Error handling was inconsistent
  console.error(error);
}
```

**After:**
```typescript
import { getData } from './client';
import { useApiErrorHandler } from '../components/api/ApiErrorHandler';

const { handleApiError } = useApiErrorHandler();

try {
  const newUser = await getData(apiClient.post('/users', userData));
  // Success - user created
} catch (error) {
  handleApiError(error);  // Toast shown, auth handled
}
```

### Example 3: Conditional Error Handling

**Before:**
```typescript
const response = await fetch('/admin', {
  headers: { Authorization: `Bearer ${token}` }
});
if (response.status === 204) {
  return true;
} else if (response.status === 401) {
  // Manual auth handling
  return false;
}
```

**After:**
```typescript
import { APIError } from './errors';

try {
  const { response } = await apiClient.fetchResponse('/admin', {
    method: 'GET'
  });
  return response.status === 204;
} catch (error) {
  if (error instanceof APIError && error.status === 401) {
    return false;
  }
  throw error;
}
```

### Example 4: Search with Query Parameters

**Before:**
```typescript
const params = new URLSearchParams();
params.append('q', searchTerm);
params.append('page', '1');

const results = await fetch(`/search?${params}`, {
  headers: { Authorization: `Bearer ${token}` }
})
  .then(checkStatus)
  .then(r => r.json());
```

**After:**
```typescript
const { data } = await apiClient.post('/search', {
  search_term: searchTerm,
  page: 1
});

// Or for GET requests:
const { data } = await apiClient.get('/search', {
  params: { q: searchTerm, page: 1 }
});
```

## Testing API Calls

### Testing Success Cases

```typescript
import { describe, it, expect, vi } from 'vitest';
import { apiClient } from './client';

// Mock fetch
global.fetch = vi.fn();

it('should fetch user data', async () => {
  const mockUser = { id: 1, name: 'Test' };
  (global.fetch as any).mockResolvedValue({
    ok: true,
    status: 200,
    json: async () => mockUser,
  });

  const { data } = await apiClient.get('/users/1');
  expect(data).toEqual(mockUser);
});
```

### Testing Error Cases

```typescript
import { AuthError, NetworkError } from './errors';

it('should handle auth errors', async () => {
  (global.fetch as any).mockResolvedValue({
    ok: false,
    status: 401,
    text: async () => 'Unauthorized',
  });

  await expect(apiClient.get('/protected')).rejects.toThrow(AuthError);
});

it('should handle network errors', async () => {
  (global.fetch as any).mockRejectedValue(new TypeError('Failed to fetch'));

  await expect(apiClient.get('/endpoint')).rejects.toThrow(NetworkError);
});
```

### Testing with Error Handler

```typescript
import { renderHook, act } from '@testing-library/react';
import { useApiErrorHandler } from '../components/api/ApiErrorHandler';

it('should handle auth errors correctly', async () => {
  const { result } = renderHook(() => useApiErrorHandler());
  const authError = new AuthError('Test error');

  // Mock the dependencies
  // ... setup mocks for logout, toast, navigate

  act(() => {
    result.current.handleApiError(authError);
  });

  // Assert logout was called, toast shown, redirect happened
});
```

## Migrating Existing API Modules

If you have API modules that use the old `checkStatus` pattern:

1. **Replace imports:**
   ```typescript
   // Old
   import { checkStatus } from './common';

   // New
   import { apiClient, getData } from './client';
   ```

2. **Update fetch calls:**
   ```typescript
   // Old
   return fetch(url, { headers })
     .then(checkStatus)
     .then(r => r.json());

   // New
   return getData(apiClient.get(url));
   ```

3. **Add error handling in components:**
   ```typescript
   const { handleApiError } = useApiErrorHandler();

   try {
     const data = await myApiFunction();
   } catch (error) {
     handleApiError(error);
   }
   ```

## Best Practices

1. **Always use the error handler** in components for user-facing API calls
2. **Use type guards** when you need different handling for different error types
3. **Test error cases** - the new system makes this much easier
4. **Use `getData` helper** for most cases - destructure only when you need the response
5. **Don't swallow errors** - let the error handler deal with them or handle explicitly

## Troubleshooting

### "AuthError is not defined"
Make sure to import error types from './errors':
```typescript
import { AuthError } from './errors';
```

### "handleApiError is not a function"
Ensure you're calling the hook correctly:
```typescript
const { handleApiError } = useApiErrorHandler();
```

### Tests failing after migration
You may need to update mocks to return Response objects or throw typed errors instead of relying on side effects.

## Backward Compatibility

The old `checkStatus` function still exists in `common.ts` for backward compatibility, but it now throws proper error types instead of triggering side effects. This means existing code will continue to work, but you should migrate to the new pattern for:
- Better testability
- Consistent error handling
- No unexpected side effects
- Better TypeScript support

## Summary

| Aspect | Old Pattern | New Pattern |
|--------|-------------|-------------|
| Error handling | Side effects (reload) | Typed errors |
| Testing | Difficult | Easy |
| Type safety | Limited | Full TypeScript support |
| Separation of concerns | Mixed | Clean separation |
| Toast notifications | Manual | Automatic with hook |
| Auth redirects | In API layer | In UI layer |

For questions or issues, refer to the test files in `src/api/*.test.ts` for examples.
