import type { InputProps } from '../../types'
import './Input.css'

export const Input = ({
  label,
  value,
  onChange,
  type = 'text',
  error,
  placeholder
}: InputProps) => {
  const inputId = `input-${label.toLowerCase().replace(/\s+/g, '-')}`
  
  return (
    <div className="input-group">
      <label htmlFor={inputId} className="input-label">
        {label}
      </label>
      <input
        id={inputId}
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className={`input ${error ? 'input--error' : ''}`}
        aria-describedby={error ? `${inputId}-error` : undefined}
        aria-invalid={error ? 'true' : 'false'}
      />
      {error && (
        <div id={`${inputId}-error`} className="input-error" role="alert">
          {error}
        </div>
      )}
    </div>
  )
}