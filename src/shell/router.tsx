import { createBrowserRouter } from 'react-router-dom';
import { AppLayout } from './AppLayout';
import { CanvasTerminalRoute } from './CanvasTerminalRoute';
import { NotFoundPage } from './NotFoundPage';

// eslint-disable-next-line agentverse/no-sideways-capability-imports
import { ProvidersPage, GeneralPage, AppearancePage } from '@/settings/routes';

// eslint-disable-next-line agentverse/no-sideways-capability-imports
import { CanvasListPage } from '@/canvas-builder/CanvasListPage';
// eslint-disable-next-line agentverse/no-sideways-capability-imports
import { CanvasBuilderPage } from '@/canvas-builder/CanvasBuilderPage';
// eslint-disable-next-line agentverse/no-sideways-capability-imports
import { AgentStudioPage } from '@/agent-studio';
// eslint-disable-next-line agentverse/no-sideways-capability-imports
import { DashboardPage } from '@/dashboard';
// eslint-disable-next-line agentverse/no-sideways-capability-imports
import { HealthPage } from '@/health/HealthPage';
// eslint-disable-next-line agentverse/no-sideways-capability-imports
import { FlowsPage } from '@/flows';
// eslint-disable-next-line agentverse/no-sideways-capability-imports
import { MemoryViewerPage } from '@/memory-viewer';

export const router = createBrowserRouter([
  {
    path: '/',
    element: <AppLayout />,
    children: [
      {
        path: '',
        element: <CanvasListPage />,
      },
      {
        path: 'dashboard',
        element: <DashboardPage />,
      },
      {
        path: 'canvas/:id',
        element: <CanvasBuilderPage />,
      },
      {
        path: 'canvas/:id/terminal/:terminalId',
        element: <CanvasTerminalRoute />,
      },
      {
        path: 'agent-studio',
        element: <AgentStudioPage />,
      },
      {
        path: 'flows',
        element: <FlowsPage />,
      },
      {
        path: 'finops',
        lazy: async () => ({ Component: (await import('@/finops/FinopsPage')).FinopsPage }),
      },
      {
        path: 'memory',
        element: <MemoryViewerPage />,
      },
      {
        path: 'settings/providers',
        element: <ProvidersPage />,
      },
      {
        path: 'settings/appearance',
        element: <AppearancePage />,
      },
      {
        path: 'settings/general',
        element: <GeneralPage />,
      },
      {
        path: 'health',
        element: <HealthPage />,
      },
      {
        path: '*',
        element: <NotFoundPage />,
      },
    ],
  },
]);
export default router;
