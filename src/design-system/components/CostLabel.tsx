import React from 'react';

export interface CostLabelProps extends React.HTMLAttributes<HTMLSpanElement> {
  value: string | number;
}

// Re-exported glyph helper per spec contract.
// eslint-disable-next-line react-refresh/only-export-components
export const CostWarningGlyph = '\u26a0\ufe0f';
const COST_ESTIMATE_DISCLAIMER =
  'Rough estimate based on active time. Actual costs may differ significantly. See provider dashboard for real billing.';

export const CostLabel: React.FC<CostLabelProps> = ({
  value,
  className = '',
  style,
  ...props
}) => {
  return (
    <span
      className={`sentinel-cost-label ${className}`}
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: '4px',
        fontFamily: 'var(--font-mono)',
        fontWeight: 600,
        color: 'var(--amber)',
        ...style,
      }}
      title={COST_ESTIMATE_DISCLAIMER}
      {...props}
    >
      <span className="cost-warning-glyph" aria-hidden="true">
        {CostWarningGlyph}
      </span>
      <span className="cost-value">{value}</span>
    </span>
  );
};

export default CostLabel;
