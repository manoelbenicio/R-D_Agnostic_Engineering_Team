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
    <Modal isOpen={isOpen} onClose={onClose} title="Templates" actions={null}>
      <div className="templates-picker-grid">
        {TEMPLATES.map((template) => (
          <Card key={template.id} className="template-picker-card">
            <div>
              <div className="template-picker-header">
                <h3>{template.name}</h3>
                <span>{template.agent_count} agents</span>
              </div>
              <p>{template.description}</p>
              <div className="template-picker-meta">
                <span>{template.primary_edge_type}</span>
                <span>
                  <CostWarning showText /> {formatTemplateCost(template.est_cost_per_hour_usd)}/hr
                </span>
              </div>
            </div>
            <Button variant="secondary" onClick={() => handleSelect(template.id)}>
              Use Template
            </Button>
          </Card>
        ))}
      </div>
    </Modal>
  );
};

export default TemplatePicker;
