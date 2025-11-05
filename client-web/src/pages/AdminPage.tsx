import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Container } from '../components/common/Container'
import { Card } from '../components/common/Card'
import { Button } from '../components/common/Button'
import { apiClient } from '../api/client'
import type { ComponentState, GetPollResponse } from '../types'
import './AdminPage.css'

export const AdminPage = () => {
  const { pollId } = useParams<{ pollId: string }>()
  const navigate = useNavigate()
  
  const [pollState, setPollState] = useState<ComponentState<GetPollResponse>>({
    loading: true,
    error: null,
    data: null
  })
  
  const [ballotCount, setBallotCount] = useState<number>(0)
  const [adminKey, setAdminKey] = useState<string | null>(null)
  const [closingPoll, setClosingPoll] = useState(false)
  const [copySuccess, setCopySuccess] = useState(false)

  useEffect(() => {
    if (!pollId) {
      setPollState({
        loading: false,
        error: 'Poll ID is required',
        data: null
      })
      return
    }

    // Get admin key from localStorage
    const storedAdminKey = localStorage.getItem(`admin_key_${pollId}`)
    if (!storedAdminKey) {
      setPollState({
        loading: false,
        error: 'Admin access denied. Admin key not found.',
        data: null
      })
      return
    }

    setAdminKey(storedAdminKey)
    loadPollData(pollId, storedAdminKey)
  }, [pollId])

  const loadPollData = async (pollId: string, adminKey: string) => {
    try {
      setPollState({ loading: true, error: null, data: null })
      
      // First load poll data
      const pollResponse = await apiClient.getPollAdmin(pollId, adminKey)
      
      // Then load ballot count using the poll slug
      const ballotResponse = await apiClient.getBallotCount(pollResponse.poll.slug)

      setPollState({
        loading: false,
        error: null,
        data: pollResponse
      })
      
      setBallotCount(ballotResponse.ballot_count)
    } catch (error) {
      setPollState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to load poll data',
        data: null
      })
    }
  }

  const handleClosePoll = async () => {
    if (!pollId || !adminKey || !pollState.data) return

    setClosingPoll(true)
    try {
      await apiClient.closePoll(pollId, { admin_key: adminKey })
      
      // Reload poll data to reflect the closed status
      await loadPollData(pollId, adminKey)
    } catch (error) {
      setPollState({
        ...pollState,
        error: error instanceof Error ? error.message : 'Failed to close poll'
      })
    } finally {
      setClosingPoll(false)
    }
  }

  const handleCopyLink = async () => {
    if (!pollState.data?.poll.slug) return

    const pollUrl = `${window.location.origin}/poll/${pollState.data.poll.slug}`
    
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
    if (pollState.data?.poll.slug) {
      navigate(`/poll/${pollState.data.poll.slug}/results`)
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

  if (pollState.loading) {
    return (
      <Container>
        <div className="admin-page">
          <div className="loading-message">Loading poll data...</div>
        </div>
      </Container>
    )
  }

  if (pollState.error) {
    return (
      <Container>
        <div className="admin-page">
          <Card>
            <h1>Admin Access Error</h1>
            <div className="error-message" role="alert">
              {pollState.error}
            </div>
            <div className="admin-actions">
              <Button onClick={() => navigate('/create')}>
                Create New Poll
              </Button>
            </div>
          </Card>
        </div>
      </Container>
    )
  }

  const poll = pollState.data!.poll
  const options = pollState.data!.options

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
          <p>Share this link with voters to allow them to participate:</p>
          
          <div className="share-link">
            <code className="poll-url">
              {window.location.origin}/poll/{poll.slug}
            </code>
            <Button onClick={handleCopyLink}>
              {copySuccess ? 'Copied!' : 'Copy Link'}
            </Button>
          </div>
        </Card>

        <Card>
          <h3>Poll Management</h3>
          <div className="admin-actions">
            {poll.status === 'open' && (
              <Button 
                onClick={handleClosePoll}
                disabled={closingPoll}
              >
                {closingPoll ? 'Closing Poll...' : 'Close Poll'}
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