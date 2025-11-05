import type { InputProps } from '../../types'
import { InlineError } from './InlineError'
import './Input.css'

interface ExtendedInputProps extends InputProps {
  required?: boolean
  disabled?: boolean
}

export const Input = ({
  label,
  value,
  onChange,
  type = 'text',
  error,
  placeholder,
  required = false,
  disabled = false
}: ExtendedInputProps) => {
  const inputId = `input-${label.toLowerCase().replace(/\s+/g, '-')}`
  const errorId = error ? `${inputId}-error` : undefined
  
  return (
    <div className="input-group">
      <label htmlFor={inputId} className="input-label">
        {label}
        {required && <span style={{ color: '#d32f2f' }}> *</span>}
      </label>
      <input
        id={inputId}
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className={`input ${error ? 'input--error' : ''}`}
        aria-describedby={errorId}
        aria-invalid={error ? 'true' : 'false'}
        required={required}
        disabled={disabled}
      />
      {error && (
        <InlineError message={error} id={errorId} />
      )}
    </div>
  )
}