import React from 'react';

export interface FormFieldProps extends React.HTMLAttributes<HTMLDivElement> {
  label: string;
  id: string;
  helperText?: string;
  errorText?: string;
  required?: boolean;
  children: React.ReactElement;
}

export const FormField: React.FC<FormFieldProps> = ({
  label,
  id,
  helperText,
  errorText,
  required = false,
  children,
  className = '',
  style,
  ...props
}) => {
  const isError = !!errorText;
  
  const child = React.cloneElement(children, {
    id,
    'aria-describedby': isError ? `${id}-error` : helperText ? `${id}-helper` : undefined,
    'aria-invalid': isError ? 'true' : undefined,
    required: required || children.props.required,
    className: `sentinel-input-element ${isError ? 'input-error' : ''} ${children.props.className || ''}`,
    style: {
      width: '100%',
      padding: 'var(--space-2) var(--space-3)',
      background: 'var(--surface-input)',
      border: `1px solid ${isError ? 'var(--threat)' : 'var(--border)'}`,
      borderRadius: 'var(--radius-button)',
      color: 'var(--text-primary)',
      fontFamily: 'var(--font-mono)',
      fontSize: '0.875rem',
      outline: 'none',
      boxSizing: 'border-box',
      marginTop: 'var(--space-1)',
      transition: 'all 0.15s ease-in-out',
      ...children.props.style,
    },
  });

  const fieldStyle: React.CSSProperties = {
    display: 'flex',
    flexDirection: 'column',
    marginBottom: 'var(--space-4)',
    width: '100%',
    ...style,
  };

  const labelStyle: React.CSSProperties = {
    fontSize: '0.75rem',
    fontFamily: 'var(--font-mono)',
    fontWeight: 700,
    color: isError ? 'var(--threat)' : 'var(--text-muted)',
    letterSpacing: '0.05em',
    textTransform: 'uppercase',
    display: 'flex',
    alignItems: 'center',
    gap: '4px',
  };

  return (
    <div className={`sentinel-form-field ${className}`} style={fieldStyle} {...props}>
      <label htmlFor={id} style={labelStyle}>
        {label}
        {required && <span style={{ color: 'var(--threat)' }}>*</span>}
      </label>
      
      {child}
      
      {isError && (
        <span
          id={`${id}-error`}
          className="sentinel-field-error"
          style={{
            fontSize: '0.75rem',
            fontFamily: 'var(--font-mono)',
            color: 'var(--threat)',
            marginTop: 'var(--space-1)',
          }}
        >
          {errorText}
        </span>
      )}
      
      {!isError && helperText && (
        <span
          id={`${id}-helper`}
          className="sentinel-field-helper"
          style={{
            fontSize: '0.75rem',
            fontFamily: 'var(--font-mono)',
            color: 'var(--text-dim)',
            marginTop: 'var(--space-1)',
          }}
        >
          {helperText}
        </span>
      )}
    </div>
  );
};

export default FormField;
