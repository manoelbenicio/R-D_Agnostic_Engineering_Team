import React from 'react';
import Badge from './Badge';

export interface StatusBadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  status?: 'idle' | 'processing' | 'completed' | 'waiting_user_answer' | 'error';
  label?: string;
}

export const StatusBadge: React.FC<StatusBadgeProps> = ({
  status = 'idle',
  label,
  className = '',
  ...props
}) => {
  const getGlyph = () => {
    switch (status) {
      case 'processing':
        return '●';
      case 'completed':
        return '✓';
      case 'waiting_user_answer':
        return '⚠';
      case 'error':
        return '✕';
      case 'idle':
      default:
        return '○';
    }
  };

  const getLabel = () => {
    if (label) return label;
    switch (status) {
      case 'processing':
        return 'Processing';
      case 'completed':
        return 'Completed';
      case 'waiting_user_answer':
        return 'Waiting';
      case 'error':
        return 'Error';
      case 'idle':
      default:
        return 'Idle';
    }
  };

  const glyphClass = status === 'processing' ? 'blink' : '';

  return (
    <Badge
      variant={status}
      className={`sentinel-status-badge ${className}`}
      {...props}
    >
      <span className={`status-glyph ${glyphClass}`} style={{ marginRight: '4px' }}>
        {getGlyph()}
      </span>
      <span className="status-label">{getLabel()}</span>
    </Badge>
  );
};

export default StatusBadge;
