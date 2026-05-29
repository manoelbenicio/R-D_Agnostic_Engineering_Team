/* eslint-disable agentverse/no-sideways-capability-imports -- AppLayout is the central shell layout and orchestrator, which needs to cross boundaries to initialize settings, canvas stores, and the onboarding wizard. */
import React, { useEffect, useState } from 'react';
import { Outlet } from 'react-router-dom';
import { NavBar } from './NavBar';
import { ErrorBoundary } from './ErrorBoundary';
import { ToastContainer } from './ToastContainer';
import { dbGet, dbPut } from '@/shared/storage/idb';
import { useKeyStore } from '@/api/key-store/store';
import { useSettingsStore } from '@/settings/settings-store';
import { canvasStore } from '@/canvas-document/store';
import { FirstRunWizard } from '@/health/FirstRunWizard';
import { useHealthStore } from '@/api';
import { useDataAnimateObserver } from '@/design-system';

export const AppLayout: React.FC<{ children?: React.ReactNode }> = ({ children }) => {
  // DSS motion infrastructure — reveals [data-animate] descendants on scroll.
  useDataAnimateObserver();

  const initKeys = useKeyStore((s) => s.init);
  const keysInitialized = useKeyStore((s) => s.initialized);
  const initSettings = useSettingsStore((s) => s.init);
  const settingsInitialized = useSettingsStore((s) => s.initialized);

  const [showWizard, setShowWizard] = useState(false);
  const [checkingWizard, setCheckingWizard] = useState(true);

  useEffect(() => {
    const { start, stop } = useHealthStore.getState();
    start();
    return () => {
      stop();
    };
  }, []);

  useEffect(() => {
    async function checkFirstRun() {
      if (!keysInitialized) {
        await initKeys();
      }
      if (!settingsInitialized) {
        await initSettings();
      }

      try {
        const wizardState = await dbGet('app_state', 'wizard_completed');
        if (wizardState?.value === true) {
          setShowWizard(false);
          setCheckingWizard(false);
          return;
        }

        // Subsequent visits check
        const canvases = await canvasStore.list();
        const currentValidated = useKeyStore.getState().validated;

        if (canvases.length > 0 && currentValidated.length > 0) {
          await dbPut('app_state', { key: 'wizard_completed', value: true });
          setShowWizard(false);
        } else {
          setShowWizard(true);
        }
      } catch (err) {
        console.error('Failed to check onboarding wizard status:', err);
      } finally {
        setCheckingWizard(false);
      }
    }
    void checkFirstRun();
  }, [initKeys, keysInitialized, initSettings, settingsInitialized]);

  return (
    <div className="sentinel-app-layout">
      <NavBar />
      <main className="sentinel-main-content">
        <ErrorBoundary>
          {children ?? <Outlet />}
        </ErrorBoundary>
      </main>
      <ToastContainer />

      {!checkingWizard && showWizard && (
        <FirstRunWizard onClose={() => setShowWizard(false)} />
      )}
    </div>
  );
};
export default AppLayout;
