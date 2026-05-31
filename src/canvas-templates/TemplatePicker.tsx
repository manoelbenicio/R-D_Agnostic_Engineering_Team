import React from 'react';
import { Button, Card, Modal } from '@/design-system';
// eslint-disable-next-line agentverse/no-sideways-capability-imports
import { CostWarning } from '@/finops/cost-warning';
import { CanvasDocument } from '@/shared/canvas-types';
import { formatTemplateCost, instantiateTemplate, TEMPLATES } from './templates';

export interface TemplatePickerProps {
  isOpen: boolean;
  onClose: () => void;
  onSelect: (doc: CanvasDocument) => void;
}

const gridStyle: React.CSSProperties = {
  display: 'flex',
  flexDirection: 'column',
  gap: '16px',
};

const cardHeaderStyle: React.CSSProperties = {
  display: 'flex',
  justifyContent: 'space-between',
  alignItems: 'baseline',
  marginBottom: '4px',
};

const cardTitleStyle: React.CSSProperties = {
  margin: 0,
  fontSize: '1rem',
  fontWeight: 700,
  color: 'var(--cyan)',
  fontFamily: 'var(--font-mono)',
};

const cardCountStyle: React.CSSProperties = {
  fontSize: '0.8rem',
  color: 'var(--text-muted)',
  fontFamily: 'var(--font-mono)',
};

const cardDescStyle: React.CSSProperties = {
  fontSize: '0.85rem',
  color: 'var(--text-secondary)',
  margin: '4px 0 8px',
  lineHeight: 1.4,
};

const cardMetaStyle: React.CSSProperties = {
  display: 'flex',
  gap: '16px',
  fontSize: '0.8rem',
  color: 'var(--text-muted)',
  fontFamily: 'var(--font-mono)',
  marginBottom: '12px',
};

export const TemplatePicker: React.FC<TemplatePickerProps> = ({
  isOpen,
  onClose,
  onSelect,
}) => {
  const handleSelect = (templateId: string) => {
    onSelect(instantiateTemplate(templateId));
    onClose();
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Templates" actions={null} style={{ maxWidth: '720px' }}>
      <div style={gridStyle}>
        {TEMPLATES.map((template) => (
          <Card key={template.id} style={{ padding: '16px' }}>
            <div style={cardHeaderStyle}>
              <h3 style={cardTitleStyle}>{template.name}</h3>
              <span style={cardCountStyle}>{template.agent_count} agents</span>
            </div>
            <p style={cardDescStyle}>{template.description}</p>
            <div style={cardMetaStyle}>
              <span>{template.primary_edge_type}</span>
              <span>
                <CostWarning showText /> {formatTemplateCost(template.est_cost_per_hour_usd)}/hr
              </span>
            </div>
            <Button variant="secondary" onClick={() => handleSelect(template.id)} style={{ width: '100%' }}>
              Use Template
            </Button>
          </Card>
        ))}
      </div>
    </Modal>
  );
};

export default TemplatePicker;

