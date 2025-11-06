import { Button } from './Button'

interface ErrorMessageProps {
  title?: string
  message: string
  onRetry?: () => void
  retryLabel?: string
  showRetry?: boolean
}

export function ErrorMessage({ 
  title = 'Error',
  message, 
  onRetry, 
  retryLabel = 'Try Again',
  showRetry = true 
}: ErrorMessageProps) {
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
        margin: '0 0 8px 0', 
        color: '#d32f2f',
        fontSize: '18px'
      }}>
        {title}
      </h3>
      <p style={{ 
        margin: '0 0 16px 0', 
        color: '#d32f2f'
      }}>
        {message}
      </p>
      {showRetry && onRetry && (
        <Button onClick={onRetry}>
          {retryLabel}
        </Button>
      )}
    </div>
  )
}