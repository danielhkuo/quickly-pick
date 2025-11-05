import { InlineError } from './InlineError'

interface FormValidationProps {
  errors: Record<string, string>
  generalError?: string
}

export function FormValidation({ errors, generalError }: FormValidationProps) {
  const hasErrors = Object.keys(errors).length > 0 || generalError

  if (!hasErrors) return null

  return (
    <div 
      style={{ 
        padding: '16px', 
        border: '2px solid #d32f2f', 
        backgroundColor: '#ffebee',
        borderRadius: '4px',
        marginBottom: '16px'
      }}
      role="alert"
    >
      <h3 style={{ 
        margin: '0 0 12px 0', 
        color: '#d32f2f',
        fontSize: '16px'
      }}>
        Please fix the following errors:
      </h3>
      
      {generalError && (
        <div style={{ marginBottom: '8px' }}>
          <InlineError message={generalError} />
        </div>
      )}
      
      {Object.entries(errors).map(([field, message]) => (
        <div key={field} style={{ marginBottom: '4px' }}>
          <InlineError message={`${field}: ${message}`} />
        </div>
      ))}
    </div>
  )
}