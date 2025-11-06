import { useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Container, Card, Button, LoadingSpinner, ErrorMessage, NetworkError } from '../components/common'
import { useAsyncOperation } from '../hooks/useAsyncOperation'
import { apiClient } from '../api'
import type { Poll, OptionRanking } from '../types'

export const ResultsPage = () => {
  const { slug } = useParams<{ slug: string }>()
  const navigate = useNavigate()
  
  const resultsOperation = useAsyncOperation<{
    poll: Poll
    rankings: OptionRanking[]
    ballot_count: number
  }>()

  // Load results data
  useEffect(() => {
    if (!slug) return

    // Load results data directly in useEffect to avoid dependency issues
    resultsOperation.execute(async () => {
      return await apiClient.getResults(slug)
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [slug]) // Only depend on slug - resultsOperation is intentionally excluded to prevent infinite loops

  // Navigation handlers
  const handleBackToPoll = () => {
    if (slug) {
      navigate(`/poll/${slug}`)
    }
  }

  const handleCreateNewPoll = () => {
    navigate('/create')
  }

  // Helper function to format date
  const formatDate = (dateString: string) => {
    const date = new Date(dateString)
    return date.toLocaleString()
  }

  // Helper function to format percentage
  const formatPercentage = (value: number) => {
    return `${(value * 100).toFixed(1)}%`
  }

  // Helper function to get semantic rating label
  const getRatingLabel = (value: number) => {
    if (value <= 0.1) return 'Strongly dislike'
    if (value <= 0.3) return 'Dislike'
    if (value <= 0.7) return 'Neutral'
    if (value <= 0.9) return 'Like'
    return 'Strongly like'
  }

  // Loading state
  if (resultsOperation.state.loading) {
    return <LoadingSpinner message="Loading results..." />
  }

  // Error state
  if (resultsOperation.state.error || !resultsOperation.state.data) {
    // Check if it's a network error
    if (resultsOperation.state.error?.includes('connect') || resultsOperation.state.error?.includes('network')) {
      return (
        <NetworkError 
          message={resultsOperation.state.error}
          onRetry={() => window.location.reload()}
          onGoHome={() => navigate('/')}
        />
      )
    }

    return (
      <Container>
        <Card>
          <ErrorMessage
            title="Results Not Available"
            message={resultsOperation.state.error || 'The poll results could not be loaded.'}
            onRetry={() => window.location.reload()}
            retryLabel="Reload Page"
          />
          <Button onClick={() => navigate('/')} fullWidth>
            Return to Home
          </Button>
        </Card>
      </Container>
    )
  }

  const { poll, rankings, ballot_count } = resultsOperation.state.data

  // Sealed results view for open polls
  if (poll.status === 'open') {
    return (
      <Container>
        <Card>
          <h1>{poll.title}</h1>
          <h2>Results Sealed</h2>
          <p>
            This poll is still open for voting. Results will be available once the poll is closed.
          </p>
          
          <div style={{ marginTop: '16px', marginBottom: '24px' }}>
            <p><strong>Current vote count:</strong> {ballot_count}</p>
            <p><strong>Poll created by:</strong> {poll.creator_name}</p>
            <p><strong>Created:</strong> {formatDate(poll.created_at)}</p>
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <Button onClick={handleBackToPoll} fullWidth>
              Back to Poll
            </Button>
            <Button onClick={handleCreateNewPoll} fullWidth>
              Create New Poll
            </Button>
          </div>
        </Card>
      </Container>
    )
  }

  // Results display for closed polls
  return (
    <Container>
      <Card>
        <h1>{poll.title}</h1>
        <h2>Final Results</h2>
        
        {poll.description && (
          <p style={{ marginBottom: '16px' }}>{poll.description}</p>
        )}

        {/* Poll metadata */}
        <div style={{ marginBottom: '32px', padding: '16px', border: '1px solid', backgroundColor: 'rgba(0, 0, 0, 0.02)' }}>
          <h3>Poll Information</h3>
          <p><strong>Created by:</strong> {poll.creator_name}</p>
          <p><strong>Created:</strong> {formatDate(poll.created_at)}</p>
          {poll.closed_at && (
            <p><strong>Closed:</strong> {formatDate(poll.closed_at)}</p>
          )}
          <p><strong>Total votes:</strong> {ballot_count}</p>
        </div>

        {/* Rankings display */}
        <div style={{ marginBottom: '32px' }}>
          <h3>Rankings (Balanced Majority Judgment)</h3>
          
          {rankings.length === 0 ? (
            <p>No votes were cast for this poll.</p>
          ) : (
            <div>
              {rankings.map((ranking) => (
                <div 
                  key={ranking.option_id} 
                  style={{ 
                    marginBottom: '24px', 
                    padding: '20px', 
                    border: '2px solid',
                    borderColor: ranking.veto ? '#d32f2f' : 'inherit',
                    backgroundColor: ranking.veto ? '#ffebee' : 'rgba(0, 0, 0, 0.02)'
                  }}
                >
                  <div style={{ display: 'flex', alignItems: 'center', marginBottom: '12px' }}>
                    <h4 style={{ margin: 0, marginRight: '16px' }}>
                      #{ranking.rank} {ranking.label}
                    </h4>
                    {ranking.veto && (
                      <span style={{ 
                        color: '#d32f2f', 
                        fontWeight: 'bold',
                        fontSize: '14px',
                        padding: '4px 8px',
                        border: '1px solid #d32f2f',
                        borderRadius: '4px'
                      }}>
                        VETOED
                      </span>
                    )}
                  </div>

                  <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '16px', marginBottom: '12px' }}>
                    <div>
                      <strong>Median:</strong> {getRatingLabel(ranking.median)} ({ranking.median.toFixed(2)})
                    </div>
                    <div>
                      <strong>Mean:</strong> {getRatingLabel(ranking.mean)} ({ranking.mean.toFixed(2)})
                    </div>
                    <div>
                      <strong>10th percentile:</strong> {getRatingLabel(ranking.p10)} ({ranking.p10.toFixed(2)})
                    </div>
                    <div>
                      <strong>90th percentile:</strong> {getRatingLabel(ranking.p90)} ({ranking.p90.toFixed(2)})
                    </div>
                  </div>

                  <div style={{ marginBottom: '8px' }}>
                    <strong>Negative votes:</strong> {formatPercentage(ranking.neg_share)}
                  </div>

                  {ranking.veto && (
                    <div style={{ 
                      marginTop: '12px', 
                      padding: '12px', 
                      backgroundColor: '#ffcdd2',
                      border: '1px solid #d32f2f',
                      borderRadius: '4px'
                    }}>
                      <strong>Veto Explanation:</strong> This option received more than 33% negative votes 
                      ({formatPercentage(ranking.neg_share)}) and has been vetoed. 
                      Vetoed options cannot win regardless of their median score.
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Explanation of BMJ */}
        <div style={{ marginBottom: '32px', padding: '16px', border: '1px solid', backgroundColor: 'rgba(0, 0, 0, 0.02)' }}>
          <h3>About Balanced Majority Judgment</h3>
          <p>
            Rankings are determined by the median rating each option received. 
            Options with higher median ratings rank higher. If options have the same median, 
            the percentile ranges are used as tiebreakers.
          </p>
          <p>
            <strong>Veto Rule:</strong> Any option that receives more than 33% negative votes 
            (ratings below neutral) is automatically vetoed and cannot win, regardless of its median score.
          </p>
        </div>

        {/* Navigation */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
          <Button onClick={handleBackToPoll} fullWidth>
            Back to Poll
          </Button>
          <Button onClick={handleCreateNewPoll} fullWidth>
            Create New Poll
          </Button>
        </div>
      </Card>
    </Container>
  )
}