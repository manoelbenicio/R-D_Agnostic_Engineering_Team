import React from 'react';
import { Card } from '@/design-system';
import { ROLE_TEMPLATES, StarterRole } from './role-templates';

const STARTER_BLOCKS: Array<{ role: StarterRole; glyph: string }> = [
  { role: 'supervisor', glyph: 'S' },
  { role: 'developer', glyph: 'D' },
  { role: 'reviewer', glyph: 'R' },
  { role: 'custom', glyph: '+' },
];

export const AgentPalette: React.FC = () => {
  const handleDragStart = (event: React.DragEvent<HTMLDivElement>, role: StarterRole) => {
    event.dataTransfer.setData('application/agentverse-role', role);
    event.dataTransfer.effectAllowed = 'move';
  };

  return (
    <aside className="canvas-palette" aria-label="Agent palette">
      <div className="canvas-panel-title">Agent Palette</div>
      <div className="canvas-palette-grid">
        {STARTER_BLOCKS.map((block) => {
          const template = ROLE_TEMPLATES[block.role];
          return (
            <Card
              key={block.role}
              className="canvas-palette-item"
              draggable
              onDragStart={(event) => handleDragStart(event, block.role)}
              tabIndex={0}
              role="button"
              aria-label={`Drag ${template.display_name} block`}
            >
              <span className="canvas-palette-glyph">{block.glyph}</span>
              <span>{template.display_name}</span>
            </Card>
          );
        })}
      </div>
    </aside>
  );
};

export default AgentPalette;
