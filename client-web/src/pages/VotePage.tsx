import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Container } from '../components/common/Container'
import { Card } from '../components/common/Card'
import { Input } from '../components/common/Input'
import { Button } from '../components/common/Button'
import { Slider } from '../components/common/Slider'
import { apiClient } from '../api'
import type { Poll, Option, ComponentState } from '../types'

type VotePageState = ComponentState<{ poll: Poll; options: Option[] }>

interface VotingState {
  username: string
  voterToken: string | null
  ratings: Record<string, number>
  hasVoted: boolean
}

export const VotePage = () => {
  const { slug } = useParams<{ slug: string }>()
  const navigate = useNavigate()
  
  // Poll data state
  const [pollState, setPollState] = useState<VotePageState>({
    loading: true,
    error: null,
    data: null
  })
  
  // Voting state
  const [votingState, setVotingState] = useState<VotingState>({
    username: '',
    voterToken: null,
    ratings: {},
    hasVoted: false
  })
  
  // Form states
  const [usernameError, setUsernameError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [submitError, setSubmitError] = useState<string | null>(null)

  // Load poll data and check for existing voter session
  useEffect(() => {
    if (!slug) return

    const loadPollData = async () => {
      try {
        setPollState({ loading: true, error: null, data: null })
        const response = await apiClient.getPoll(slug)
        
        // Initialize ratings with neutral values (0.5)
        const initialRatings: Record<string, number> = {}
        response.options.forEach(option => {
          initialRatings[option.id] = 0.5
        })
        
        setPollState({ loading: false, error: null, data: response })
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
      } catch (error) {
        setPollState({
          loading: false,
          error: error instanceof Error ? error.message : 'Failed to load poll',
          data: null
        })
      }
    }

    loadPollData()
  }, [slug])

  // Handle username claim
  const handleClaimUsername = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!slug || !votingState.username.trim()) return

    try {
      setSubmitting(true)
      setUsernameError(null)
      
      const response = await apiClient.claimUsername(slug, {
        username: votingState.username.trim()
      })
      
      // Store credentials in localStorage
      localStorage.setItem(`voter_token_${slug}`, response.voter_token)
      localStorage.setItem(`username_${slug}`, votingState.username.trim())
      
      setVotingState(prev => ({
        ...prev,
        voterToken: response.voter_token,
        username: votingState.username.trim()
      }))
    } catch (error) {
      if (error instanceof Error) {
        setUsernameError(error.message)
      } else {
        setUsernameError('Failed to claim username')
      }
    } finally {
      setSubmitting(false)
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
  const handleSubmitBallot = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!slug || !votingState.voterToken || !pollState.data) return

    try {
      setSubmitting(true)
      setSubmitError(null)
      
      await apiClient.submitBallot(slug, {
        voter_token: votingState.voterToken,
        ratings: votingState.ratings
      })
      
      setVotingState(prev => ({ ...prev, hasVoted: true }))
    } catch (error) {
      if (error instanceof Error) {
        setSubmitError(error.message)
      } else {
        setSubmitError('Failed to submit ballot')
      }
    } finally {
      setSubmitting(false)
    }
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
  if (pollState.loading) {
    return (
      <Container>
        <Card>
          <p>Loading poll...</p>
        </Card>
      </Container>
    )
  }

  // Error state
  if (pollState.error || !pollState.data) {
    return (
      <Container>
        <Card>
          <h1>Error</h1>
          <p>{pollState.error || 'Poll not found'}</p>
          <Button onClick={() => navigate('/')}>
            Return to Home
          </Button>
        </Card>
      </Container>
    )
  }

  const { poll, options } = pollState.data

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
              error={usernameError || undefined}
            />
            
            <div style={{ marginTop: '24px' }}>
              <Button 
                type="submit" 
                disabled={submitting || !votingState.username.trim()}
                fullWidth
              >
                {submitting ? 'Claiming...' : 'Claim Username'}
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

          {submitError && (
            <div style={{ 
              marginBottom: '16px', 
              padding: '12px', 
              border: '1px solid #d32f2f', 
              backgroundColor: '#ffebee',
              borderRadius: '4px'
            }}>
              <p style={{ color: '#d32f2f', margin: 0 }}>{submitError}</p>
            </div>
          )}

          <Button 
            type="submit" 
            disabled={submitting}
            fullWidth
          >
            {submitting ? 'Submitting...' : 'Submit Ballot'}
          </Button>
        </form>
      </Card>
    </Container>
  )
}