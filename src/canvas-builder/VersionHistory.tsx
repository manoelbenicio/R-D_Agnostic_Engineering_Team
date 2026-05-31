/* eslint-disable agentverse/no-sideways-capability-imports */
import React, { useEffect, useState } from 'react';
import { Button, Modal } from '@/design-system';
import { canvasStore } from '@/canvas-document/store';
import { CanvasDocument } from '@/shared/canvas-types';
import './version-history.css';

interface VersionHistoryProps {
  canvasId: string;
  isOpen: boolean;
  onClose: () => void;
  onRestore: (snapshot: CanvasDocument) => void;
}

export const VersionHistory: React.FC<VersionHistoryProps> = ({
  canvasId,
  isOpen,
  onClose,
  onRestore,
}) => {
  const [versions, setVersions] = useState<CanvasDocument[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!isOpen) return;
    let cancelled = false;
    setLoading(true);
    void canvasStore
      .listVersions(canvasId)
      .then((list) => {
        if (!cancelled) setVersions([...list].sort((a, b) => b.version - a.version));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [canvasId, isOpen]);

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Version History">
      {loading ? (
        <p className="version-history-empty">Loading snapshots…</p>
      ) : versions.length === 0 ? (
        <p className="version-history-empty">
          No saved snapshots yet. Snapshots are captured each time you Save or Deploy.
        </p>
      ) : (
        <ul className="version-history-list">
          {versions.map((snapshot) => (
            <li key={snapshot.version} className="version-history-item">
              <div className="version-history-meta">
                <strong>v{snapshot.version}</strong>
                <span>{formatTimestamp(snapshot.updated_at)}</span>
                <span className="version-history-status">{snapshot.deploy_state.status}</span>
              </div>
              <ul className="version-history-nodes">
                {snapshot.nodes.map((node) => (
                  <li key={node.id}>
                    {node.data.display_name}: {node.data.provider || '—'}
                    {node.data.model ? ` · ${node.data.model}` : ''}
                  </li>
                ))}
              </ul>
              <Button
                variant="secondary"
                onClick={() => {
                  onRestore(snapshot);
                  onClose();
                }}
              >
                Restore
              </Button>
            </li>
          ))}
        </ul>
      )}
    </Modal>
  );
};

function formatTimestamp(iso: string): string {
  const parsed = Date.parse(iso);
  if (Number.isNaN(parsed)) return iso;
  return new Intl.DateTimeFormat('en-US', { dateStyle: 'medium', timeStyle: 'short' }).format(parsed);
}

export default VersionHistory;
