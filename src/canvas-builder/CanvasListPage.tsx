/* eslint-disable agentverse/no-sideways-capability-imports */
import React, { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Badge, Button, Card } from '@/design-system';
import { canvasStore } from '@/canvas-document/store';
import { CanvasDocument } from '@/shared/canvas-types';
import { TemplatePicker } from '@/canvas-templates';
import './canvas-builder.css';

export const CanvasListPage: React.FC = () => {
  const navigate = useNavigate();
  const [canvases, setCanvases] = useState<CanvasDocument[]>([]);
  const [isLoading, setLoading] = useState(true);
  const [isTemplatePickerOpen, setTemplatePickerOpen] = useState(false);

  const sortedCanvases = useMemo(
    () =>
      [...canvases].sort(
        (a, b) => Date.parse(b.updated_at) - Date.parse(a.updated_at)
      ),
    [canvases]
  );

  useEffect(() => {
    let cancelled = false;
    async function loadCanvases() {
      setLoading(true);
      const list = await canvasStore.list();
      if (!cancelled) {
        setCanvases(list);
        setLoading(false);
      }
    }
    void loadCanvases();
    return () => {
      cancelled = true;
    };
  }, []);

  const createBlankCanvas = async () => {
    const saved = await canvasStore.save(canvasStore.createDraft());
    navigate(`/canvas/${saved.id}`);
  };

  const createFromTemplate = async (doc: CanvasDocument) => {
    const saved = await canvasStore.save(doc);
    navigate(`/canvas/${saved.id}`);
  };

  return (
    <main className="canvas-list-page">
      <header className="canvas-list-header">
        <div>
          <h1 className="canvas-page-title">Canvas List</h1>
          <p>Open a saved orchestration canvas or create a new draft.</p>
        </div>
        <div className="canvas-toolbar-actions">
          <Button variant="secondary" onClick={() => setTemplatePickerOpen(true)}>
            Templates
          </Button>
          <Button variant="primary" onClick={() => void createBlankCanvas()}>
            New Canvas
          </Button>
        </div>
      </header>

      {isLoading ? (
        <Card>Loading canvases...</Card>
      ) : sortedCanvases.length === 0 ? (
        <Card className="canvas-list-empty" glow="cyan">
          <h2>No canvases yet</h2>
          <p>Create a blank draft or start from one of the built-in templates.</p>
          <div className="canvas-toolbar-actions">
            <Button variant="secondary" onClick={() => setTemplatePickerOpen(true)}>
              Browse Templates
            </Button>
            <Button variant="primary" onClick={() => void createBlankCanvas()}>
              New Canvas
            </Button>
          </div>
        </Card>
      ) : (
        <section className="canvas-list-grid" aria-label="Saved canvases">
          {sortedCanvases.map((canvas) => (
            <Link key={canvas.id} to={`/canvas/${canvas.id}`} className="canvas-list-link">
              <Card className="canvas-list-card">
                <div className="canvas-list-card-header">
                  <h2>{canvas.name}</h2>
                  <Badge variant={canvas.deploy_state.status === 'draft' ? 'idle' : 'processing'}>
                    {canvas.deploy_state.status}
                  </Badge>
                </div>
                <div className="canvas-list-card-meta">
                  <span>{canvas.nodes.length} nodes</span>
                  <span>{canvas.edges.length} edges</span>
                  <span>v{canvas.version}</span>
                </div>
                <time>{new Date(canvas.updated_at).toLocaleString()}</time>
              </Card>
            </Link>
          ))}
        </section>
      )}

      <TemplatePicker
        isOpen={isTemplatePickerOpen}
        onClose={() => setTemplatePickerOpen(false)}
        onSelect={(doc) => void createFromTemplate(doc)}
      />
    </main>
  );
};

export default CanvasListPage;
