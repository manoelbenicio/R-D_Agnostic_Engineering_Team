# Error handling

## CAO calls

Every `CaoClient` method throws `CaoApiError` or `CaoNetworkError`.

- `CaoApiError`: HTTP response received but status >= 400. Carries
  `.status`, `.endpoint`, `.body`.
- `CaoNetworkError`: no response (DNS, connection refused, abort). Carries
  `.endpoint`, `.cause`.

Consumers (hooks, mutations) propagate the error to React Query — the UI
reads `error.message` and renders a toast or inline error.

## React render errors

The shell wraps the routed view in `<ErrorBoundary>` (task 2.4). Catches
synchronous render exceptions only. Async errors are the originating
capability's responsibility.

## Surfacing to the user

| Error class               | Surface          |
| ------------------------- | ---------------- |
| Validation (Zod)          | inline FormField error |
| `CaoApiError` (4xx)       | inline / toast with verbatim body |
| `CaoApiError` (5xx)       | toast + Health page row |
| `CaoNetworkError`         | health pill turns red, banner |
| Render exception          | `<ErrorBoundary>` fallback |
