import React from 'react';

export const COST_WARNING_GLYPH = '\u26a0\ufe0f';
export const COST_ESTIMATE_DISCLAIMER =
  'Rough estimate based on active time. Actual costs may differ significantly. See provider dashboard for real billing.';

export interface CostWarningProps extends React.HTMLAttributes<HTMLSpanElement> {
  showText?: boolean;
}

export const CostWarning: React.FC<CostWarningProps> = ({
  showText = false,
  className = '',
  ...props
}) => (
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

export default CostWarning;
