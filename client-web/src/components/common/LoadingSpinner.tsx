import { Container } from './Container'
import { Card } from './Card'

interface LoadingSpinnerProps {
  message?: string
}

export function LoadingSpinner({ message = 'Loading...' }: LoadingSpinnerProps) {
  return (
    <Container>
      <Card>
        <div style={{ 
          textAlign: 'center', 
          padding: '32px 0',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          gap: '16px'
        }}>
          <div 
            style={{
              width: '32px',
              height: '32px',
              border: '3px solid transparent',
              borderTop: '3px solid currentColor',
              borderRadius: '50%',
              animation: 'spin 1s linear infinite'
            }}
            aria-hidden="true"
          />
          <p>{message}</p>
        </div>
      </Card>
      <style>{`
        @keyframes spin {
          0% { transform: rotate(0deg); }
          100% { transform: rotate(360deg); }
        }
      `}</style>
    </Container>
  )
}