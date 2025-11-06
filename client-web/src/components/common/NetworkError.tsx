import { Container } from './Container'
import { Card } from './Card'
import { Button } from './Button'

interface NetworkErrorProps {
  message?: string
  onRetry?: () => void
  onGoHome?: () => void
}

export function NetworkError({ 
  message = 'Unable to connect to the server. Please check your internet connection and try again.',
  onRetry,
  onGoHome
}: NetworkErrorProps) {
  return (
    <Container>
      <Card>
        <h1>Connection Error</h1>
        <div style={{ 
          padding: '16px', 
          border: '2px solid #d32f2f', 
          backgroundColor: '#ffebee',
          borderRadius: '4px',
          marginBottom: '24px'
        }}>
          <p style={{ color: '#d32f2f', margin: 0 }}>
            {message}
          </p>
        </div>
        
        <div style={{ 
          display: 'flex', 
          flexDirection: 'column', 
          gap: '16px' 
        }}>
          {onRetry && (
            <Button onClick={onRetry} fullWidth>
              Try Again
            </Button>
          )}
          {onGoHome && (
            <Button onClick={onGoHome} fullWidth>
              Go to Home
            </Button>
          )}
        </div>
      </Card>
    </Container>
  )
}