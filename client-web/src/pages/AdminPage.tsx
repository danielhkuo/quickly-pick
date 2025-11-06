import { useState, useEffect, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Container, Card, Button, LoadingSpinner, ErrorMessage, NetworkError } from '../components/common'
import { useAsyncOperation } from '../hooks/useAsyncOperation'
import { apiClient } from '../api/client'
import type { GetPollResponse } from '../types'
import './AdminPage.css'

export const AdminPage = () => {
  const { pollId } = useParams<{ pollId: string }>()
  const navigate = useNavigate()
  
  // Async operations
  const pollOperation = useAsyncOperation<GetPollResponse & { ballot_count: number }>()
  const closeOperation = useAsyncOperation<void>()

  
  const [adminKey, setAdminKey] = useState<string | null>(null)
  const [copySuccess, setCopySuccess] = useState(false)

  const loadPollData = useCallback(async (pollId: string, adminKey: string) => {
    await pollOperation.execute(async () => {
      // First load poll data
      const pollResponse = await apiClient.getPollAdmin(pollId, adminKey)
      
      // Then load ballot count using the poll slug (only if poll is published)
      let ballotCount = 0
      if (pollResponse.poll.share_slug) {
        const ballotResponse = await apiClient.getBallotCount(pollResponse.poll.share_slug)
        ballotCount = ballotResponse.ballot_count
      }

      return {
        ...pollResponse,
        ballot_count: ballotCount
      }
    })
  }, [pollOperation])

  useEffect(() => {
    if (!pollId) {
      pollOperation.setError('Poll ID is required')
      return
    }

    // Get admin key from localStorage
    const storedAdminKey = localStorage.getItem(`admin_key_${pollId}`)
    if (!storedAdminKey) {
      pollOperation.setError('Admin access denied. Admin key not found.')
      return
    }

    setAdminKey(storedAdminKey)
    
    // Load poll data directly in useEffect to avoid dependency issues
    pollOperation.execute(async () => {
      // First load poll data
      const pollResponse = await apiClient.getPollAdmin(pollId, storedAdminKey)
      
      // Then load ballot count using the poll slug (only if poll is published)
      let ballotCount = 0
      if (pollResponse.poll.share_slug) {
        const ballotResponse = await apiClient.getBallotCount(pollResponse.poll.share_slug)
        ballotCount = ballotResponse.ballot_count
      }

      return {
        ...pollResponse,
        ballot_count: ballotCount
      }
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pollId]) // Only depend on pollId - pollOperation is intentionally excluded to prevent infinite loops



  const handleClosePoll = async () => {
    if (!pollId || !adminKey || !pollOperation.state.data) return

    const result = await closeOperation.execute(async () => {
      await apiClient.closePoll(pollId, adminKey)
    })

    if (result !== null) {
      // Reload poll data to reflect the closed status
      await loadPollData(pollId, adminKey)
    }
  }

  const handleCopyLink = async () => {
    if (!pollOperation.state.data?.poll.share_slug) {
      // Show error feedback if no slug available
      setCopySuccess(false)
      return
    }

    const pollUrl = `${window.location.origin}/poll/${pollOperation.state.data.poll.share_slug}`
    
    try {
      await navigator.clipboard.writeText(pollUrl)
      setCopySuccess(true)
      setTimeout(() => setCopySuccess(false), 2000)
    } catch {
      // Fallback for browsers that don't support clipboard API
      const textArea = document.createElement('textarea')
      textArea.value = pollUrl
      document.body.appendChild(textArea)
      textArea.select()
      document.execCommand('copy')
      document.body.removeChild(textArea)
      setCopySuccess(true)
      setTimeout(() => setCopySuccess(false), 2000)
    }
  }

  const handleViewResults = () => {
    if (pollOperation.state.data?.poll.share_slug) {
      navigate(`/poll/${pollOperation.state.data.poll.share_slug}/results`)
    }
  }

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString()
  }

  const getStatusDisplay = (status: string) => {
    switch (status) {
      case 'draft':
        return 'Draft'
      case 'open':
        return 'Open for Voting'
      case 'closed':
        return 'Closed'
      default:
        return status
    }
  }

  if (pollOperation.state.loading) {
    return <LoadingSpinner message="Loading poll data..." />
  }

  if (pollOperation.state.error || !pollOperation.state.data) {
    // Check if it's a network error
    if (pollOperation.state.error?.includes('connect') || pollOperation.state.error?.includes('network')) {
      return (
        <NetworkError 
          message={pollOperation.state.error}
          onRetry={() => window.location.reload()}
          onGoHome={() => navigate('/')}
        />
      )
    }

    return (
      <Container>
        <div className="admin-page">
          <Card>
            <ErrorMessage
              title="Admin Access Error"
              message={pollOperation.state.error || 'Unable to access poll administration.'}
              onRetry={() => window.location.reload()}
              retryLabel="Reload Page"
            />
            <div className="admin-actions">
              <Button onClick={() => navigate('/create')} fullWidth>
                Create New Poll
              </Button>
            </div>
          </Card>
        </div>
      </Container>
    )
  }

  const poll = pollOperation.state.data.poll
  const options = pollOperation.state.data.options
  const ballotCount = pollOperation.state.data.ballot_count

  return (
    <Container>
      <div className="admin-page">
        <h1>Poll Administration</h1>
        
        <Card>
          <div className="poll-header">
            <h2>{poll.title}</h2>
            <div className="poll-status">
              <span className={`status-badge status-${poll.status}`}>
                {getStatusDisplay(poll.status)}
              </span>
            </div>
          </div>

          {poll.description && (
            <p className="poll-description">{poll.description}</p>
          )}

          <div className="poll-metadata">
            <div className="metadata-item">
              <strong>Created by:</strong> {poll.creator_name}
            </div>
            <div className="metadata-item">
              <strong>Created:</strong> {formatDate(poll.created_at)}
            </div>
            {poll.closed_at && (
              <div className="metadata-item">
                <strong>Closed:</strong> {formatDate(poll.closed_at)}
              </div>
            )}
            <div className="metadata-item">
              <strong>Total Votes:</strong> {ballotCount}
            </div>
          </div>
        </Card>

        <Card>
          <h3>Poll Options</h3>
          <ol className="options-list">
            {options.map((option) => (
              <li key={option.id} className="option-item">
                {option.label}
              </li>
            ))}
          </ol>
        </Card>

        <Card>
          <h3>Share Poll</h3>
          {poll.share_slug ? (
            <>
              <p>Share this link with voters to allow them to participate:</p>
              <div className="share-link">
                <code className="poll-url">
                  {window.location.origin}/poll/{poll.share_slug}
                </code>
                <Button onClick={handleCopyLink}>
                  {copySuccess ? 'Copied!' : 'Copy Link'}
                </Button>
              </div>
            </>
          ) : (
            <div className="poll-notice">
              <p>Poll is being prepared...</p>
              <Button onClick={() => window.location.reload()}>
                Refresh Page
              </Button>
            </div>
          )}
        </Card>

        <Card>
          <h3>Poll Management</h3>
          
          {closeOperation.state.error && (
            <ErrorMessage 
              message={closeOperation.state.error}
              onRetry={handleClosePoll}
              retryLabel="Try Closing Again"
            />
          )}
          
          <div className="admin-actions">
            {poll.status === 'open' && (
              <Button 
                onClick={handleClosePoll}
                disabled={closeOperation.state.loading}
              >
                {closeOperation.state.loading ? 'Closing Poll...' : 'Close Poll'}
              </Button>
            )}
            
            <Button onClick={handleViewResults}>
              View Results
            </Button>
            
            <Button onClick={() => navigate('/create')}>
              Create New Poll
            </Button>
          </div>
        </Card>

        {poll.status === 'open' && (
          <Card>
            <div className="poll-notice">
              <h4>Poll is Currently Open</h4>
              <p>
                Voters can still submit ballots. Close the poll when you're ready to 
                finalize results and prevent new votes.
              </p>
            </div>
          </Card>
        )}
      </div>
    </Container>
  )
}