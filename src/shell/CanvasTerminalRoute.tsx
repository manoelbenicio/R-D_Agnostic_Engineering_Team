import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
// eslint-disable-next-line agentverse/no-sideways-capability-imports
import { canvasStore } from '@/canvas-document/store';
// eslint-disable-next-line agentverse/no-sideways-capability-imports
import { TerminalGrid } from '@/terminal-grid';
import type { CanvasDocument } from '@/shared/canvas-types';

/**
 * Route component for `/canvas/:id/terminal/:terminalId`.
 *
 * Loads the parent canvas document, extracts the deployed session name, and
 * renders the real TerminalGrid. The route was previously rendering a stub
 * placeholder, which violated the v1 ship gate (no placeholders / mocks in
 * production code).
 */
export const CanvasTerminalRoute: React.FC = () => {
  const { id } = useParams<{ id: string; terminalId: string }>();
  const [canvas, setCanvas] = useState<CanvasDocument | null | undefined>(undefined);

  useEffect(() => {
    let cancelled = false;
    if (!id) {
      setCanvas(null);
      return () => {
        cancelled = true;
      };
    }
    canvasStore
      .get(id)
      .then((doc) => {
        if (!cancelled) setCanvas(doc);
      })
      .catch(() => {
        if (!cancelled) setCanvas(null);
      });
    return () => {
      cancelled = true;
    };
  }, [id]);

  if (canvas === undefined) {
    return (
      <div className="canvas-terminal-route is-loading" role="status" aria-live="polite">
        <p>Loading canvas…</p>
      </div>
    );
  }

  if (!canvas) {
    return (
      <div className="canvas-terminal-route is-error" role="alert">
        <h2>Canvas not found</h2>
        <p>
          The canvas <code>{id}</code> does not exist or its schema is incompatible.
        </p>
        <Link to="/" className="sentinel-link">
          ← Back to canvas list
        </Link>
      </div>
    );
  }

  const sessionName = canvas.deploy_state.session_name ?? canvas.config.session_name;

  if (!sessionName) {
    return (
      <div className="canvas-terminal-route is-error" role="alert">
        <h2>Canvas not deployed</h2>
        <p>
          This canvas has no active session. Open the canvas builder to deploy it.
        </p>
        <Link to={`/canvas/${id}`} className="sentinel-link">
          ← Open canvas builder
        </Link>
      </div>
    );
  }

  return <TerminalGrid sessionName={sessionName} />;
};

export default CanvasTerminalRoute;
