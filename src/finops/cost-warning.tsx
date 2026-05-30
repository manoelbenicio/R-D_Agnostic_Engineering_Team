import React from 'react';
import {
  COST_WARNING_GLYPH,
  COST_ESTIMATE_DISCLAIMER,
} from './cost-warning-constants';

export interface CostWarningProps extends React.HTMLAttributes<HTMLSpanElement> {
  showText?: boolean;
}

export function CostWarning({
  showText = false,
  className = '',
  ...props
}: CostWarningProps) {
  return (
    <span
      role="img"
      className={`finops-cost-warning ${className}`}
      title={COST_ESTIMATE_DISCLAIMER}
      aria-label={showText ? undefined : COST_ESTIMATE_DISCLAIMER}
      {...props}
    >
      <span aria-hidden="true">{COST_WARNING_GLYPH}</span>
      {showText ? <span> rough estimate</span> : null}
    </span>
  );
}

export default CostWarning;
