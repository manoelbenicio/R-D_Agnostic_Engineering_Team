import React from 'react';
import ReactDOM from 'react-dom/client';
import { RouterProvider } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { router } from './shell/router';
import { openDb } from './shared/storage/idb';
import '@/design-system/tokens.css';
import './index.css';

// Initialize IndexedDB on application startup
openDb().catch((err) => {
  console.error('Failed to initialize IndexedDB:', err);
});

// Setup global QueryClient for TanStack Query (Decision D3)
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: true,
      refetchOnReconnect: true,
      retry: 1,
    },
  },
});

async function enableMockingIfRequested() {
  if (import.meta.env.VITE_USE_MSW !== 'true') return;
  const { worker } = await import('./api/__tests__/msw/browser');
  await worker.start({
    onUnhandledRequest: 'bypass',
    serviceWorker: { url: '/mockServiceWorker.js' },
  });
}

// Observer to ensure scrollable container has keyboard access (A11y task 21.5)
if (typeof window !== 'undefined') {
  const observer = new MutationObserver(() => {
    const main = document.querySelector('.sentinel-main-content');
    if (main && !main.hasAttribute('tabindex')) {
      main.setAttribute('tabindex', '0');
      // Set inline focus outline: none if needed, or let standard CSS handle it.
      // But adding a title or aria-label can also help.
      if (!main.hasAttribute('aria-label')) {
        main.setAttribute('aria-label', 'Main Content');
      }
    }
  });
  observer.observe(document.documentElement, { childList: true, subtree: true });
}

await enableMockingIfRequested();

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  </React.StrictMode>
);
