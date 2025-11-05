import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Container, Card, Input, Button, Slider, LoadingSpinner, ErrorMessage, NetworkError } from '../components/common'
import { useAsyncOperation } from '../hooks/useAsyncOperation'
import { apiClient } from '../api'
import type { Poll, Option } from '../types'

interface VotingState {
  username: string
  voterToken: string | null
  ratings: Record<string, number>
  hasVoted: boolean
}

export const VotePage = () => {
  const { slug } = useParams<{ slug: string }>()
  const navigate = useNavigate()
  
  // Async operations
  const pollOperation = useAsyncOperation<{ poll: Poll; options: Option[] }>()
  const usernameOperation = useAsyncOperation<{ voter_token: string }>()
  const ballotOperation = useAsyncOperation<void>()
  
  // Voting state
  const [votingState, setVotingState] = useState<VotingState>({
    username: '',
    voterToken: null,
    ratings: {},
    hasVoted: false
  })

  // Load poll data and check for existing voter session
  useEffect(() => {
    if (!slug) return

    const loadPollData = async () => {
      const result = await pollOperation.execute(async () => {
        return await apiClient.getPoll(slug)
      })

      if (result) {
        // Initialize ratings with neutral values (0.5)
        const initialRatings: Record<string, number> = {}
        result.options.forEach(option => {
          initialRatings[option.id] = 0.5
        })
        
        setVotingState(prev => ({ ...prev, ratings: initialRatings }))
        
        // Check for existing voter session
        const existingToken = localStorage.getItem(`voter_token_${slug}`)
        const existingUsername = localStorage.getItem(`username_${slug}`)
        
        if (existingToken && existingUsername) {
          setVotingState(prev => ({
            ...prev,
            voterToken: existingToken,
            username: existingUsername,
            hasVoted: true
          }))
        }
      }
    }

    loadPollData()
  }, [slug, pollOperation])

  // Handle username claim
  const handleClaimUsername = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!slug || !votingState.username.trim()) return

    const result = await usernameOperation.execute(async () => {
      return await apiClient.claimUsername(slug, {
        username: votingState.username.trim()
      })
    })

    if (result) {
      // Store credentials in localStorage
      localStorage.setItem(`voter_token_${slug}`, result.voter_token)
      localStorage.setItem(`username_${slug}`, votingState.username.trim())
      
      setVotingState(prev => ({
        ...prev,
        voterToken: result.voter_token,
        username: votingState.username.trim()
      }))
    }
  }

  // Handle slider value changes
  const handleRatingChange = (optionId: string, value: number) => {
    setVotingState(prev => ({
      ...prev,
      ratings: {
        ...prev.ratings,
        [optionId]: value
      }
    }))
  }

  // Handle ballot submission
  const submitBallot = async () => {
    if (!slug || !votingState.voterToken || !pollOperation.state.data) return

    const result = await ballotOperation.execute(async () => {
      await apiClient.submitBallot(slug, {
        voter_token: votingState.voterToken!,
        ratings: votingState.ratings
      })
    })

    if (result !== null) {
      setVotingState(prev => ({ ...prev, hasVoted: true }))
    }
  }

  const handleSubmitBallot = async (e: React.FormEvent) => {
    e.preventDefault()
    await submitBallot()
  }

  // Navigate to results
  const handleViewResults = () => {
    if (slug) {
      navigate(`/poll/${slug}/results`)
    }
  }

  // Navigate to create new poll
  const handleCreateNewPoll = () => {
    navigate('/create')
  }

  // Loading state
  if (pollOperation.state.loading) {
    return <LoadingSpinner message="Loading poll..." />
  }

  // Error state
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
        <Card>
          <ErrorMessage
            title="Poll Not Found"
            message={pollOperation.state.error || 'The requested poll could not be found.'}
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

  const { poll, options } = pollOperation.state.data

  // Poll closed state
  if (poll.status === 'closed') {
    return (
      <Container>
        <Card>
          <h1>{poll.title}</h1>
          <p>This poll has been closed and is no longer accepting votes.</p>
          <div style={{ marginTop: '24px' }}>
            <Button onClick={handleViewResults} fullWidth>
              View Results
            </Button>
          </div>
        </Card>
      </Container>
    )
  }

  // Poll not open state
  if (poll.status !== 'open') {
    return (
      <Container>
        <Card>
          <h1>{poll.title}</h1>
          <p>This poll is not currently accepting votes.</p>
          <Button onClick={() => navigate('/')}>
            Return to Home
          </Button>
        </Card>
      </Container>
    )
  }

  // Voting confirmation state
  if (votingState.hasVoted && votingState.voterToken) {
    return (
      <Container>
        <Card>
          <h1>Vote Submitted!</h1>
          <p>Thank you for participating, <strong>{votingState.username}</strong>!</p>
          <p>Your ballot has been successfully recorded for "{poll.title}".</p>
          
          <div style={{ marginTop: '24px', display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <Button onClick={handleViewResults} fullWidth>
              View Results
            </Button>
            <Button onClick={handleCreateNewPoll} fullWidth>
              Create New Poll
            </Button>
          </div>
        </Card>
      </Container>
    )
  }

  // Username claim form
  if (!votingState.voterToken) {
    return (
      <Container>
        <Card>
          <h1>{poll.title}</h1>
          {poll.description && <p>{poll.description}</p>}
          
          <form onSubmit={handleClaimUsername} style={{ marginTop: '24px' }}>
            <Input
              label="Choose a username"
              value={votingState.username}
              onChange={(value) => setVotingState(prev => ({ ...prev, username: value }))}
              placeholder="Enter your username"
              error={usernameOperation.state.error || undefined}
              required
            />
            
            <div style={{ marginTop: '24px' }}>
              <Button 
                type="submit" 
                disabled={usernameOperation.state.loading || !votingState.username.trim()}
                fullWidth
              >
                {usernameOperation.state.loading ? 'Claiming...' : 'Claim Username'}
              </Button>
            </div>
          </form>
        </Card>
      </Container>
    )
  }

  // Voting interface
  return (
    <Container>
      <Card>
        <h1>{poll.title}</h1>
        {poll.description && <p>{poll.description}</p>}
        
        <div style={{ marginTop: '16px', marginBottom: '24px' }}>
          <p><strong>Voting as:</strong> {votingState.username}</p>
        </div>

        <form onSubmit={handleSubmitBallot}>
          <div style={{ marginBottom: '24px' }}>
            <h2>Rate each option:</h2>
            <p style={{ fontSize: '16px', marginBottom: '24px' }}>
              Use the sliders to rate each option from "Strongly dislike" to "Strongly like".
            </p>
            
            {options.map((option) => (
              <div key={option.id} style={{ marginBottom: '24px' }}>
                <Slider
                  label={option.label}
                  value={votingState.ratings[option.id] || 0.5}
                  onChange={(value) => handleRatingChange(option.id, value)}
                />
              </div>
            ))}
          </div>

          {ballotOperation.state.error && (
            <ErrorMessage 
              message={ballotOperation.state.error}
              onRetry={submitBallot}
              retryLabel="Try Submitting Again"
            />
          )}

          <Button 
            type="submit" 
            disabled={ballotOperation.state.loading}
            fullWidth
          >
            {ballotOperation.state.loading ? 'Submitting...' : 'Submit Ballot'}
          </Button>
        </form>
      </Card>
    </Container>
  )
}