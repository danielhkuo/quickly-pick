import { Component } from 'react'
import type { ErrorInfo, ReactNode } from 'react'
import { Container } from './Container'
import { Card } from './Card'
import { Button } from './Button'

interface Props {
  children: ReactNode
}

interface State {
  hasError: boolean
  error?: Error
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false }
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('Error boundary caught an error:', error, errorInfo)
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: undefined })
  }

  handleGoHome = () => {
    window.location.href = '/'
  }

  render() {
    if (this.state.hasError) {
      return (
        <Container>
          <Card>
            <h2>Something went wrong</h2>
            <p>
              An unexpected error occurred while loading this page. 
              You can try refreshing or return to the home page.
            </p>
            {this.state.error && (
              <details style={{ marginTop: '16px', marginBottom: '16px' }}>
                <summary>Error details</summary>
                <pre style={{ 
                  fontSize: '14px', 
                  overflow: 'auto', 
                  padding: '8px',
                  border: '1px solid',
                  marginTop: '8px'
                }}>
                  {this.state.error.message}
                </pre>
              </details>
            )}
            <div style={{ display: 'flex', gap: '16px', flexWrap: 'wrap' }}>
              <Button onClick={this.handleRetry}>
                Try Again
              </Button>
              <Button onClick={this.handleGoHome}>
                Go to Home
              </Button>
            </div>
          </Card>
        </Container>
      )
    }

    return this.props.children
  }
}