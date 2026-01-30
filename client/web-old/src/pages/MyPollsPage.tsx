import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { apiClient } from '../api/client'
import { Container } from '../components/common/Container'
import { Card } from '../components/common/Card'
import { Button } from '../components/common/Button'
import { LoadingSpinner } from '../components/common/LoadingSpinner'
import { ErrorMessage } from '../components/common/ErrorMessage'
import type { DevicePollSummary } from '../types'

export function MyPollsPage() {
  const navigate = useNavigate()
  const [polls, setPolls] = useState<DevicePollSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    async function loadPolls() {
      try {
        // Register device first (idempotent on backend)
        await apiClient.registerDevice()
        // Then fetch polls
        const data = await apiClient.getMyPolls()
        setPolls(data)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load polls')
      } finally {
        setLoading(false)
      }
    }
    loadPolls()
  }, [])

  const adminPolls = polls.filter(p => p.role === 'admin')
  const voterPolls = polls.filter(p => p.role === 'voter')

  const getStatusBadge = (status: string) => {
    const styles: Record<string, React.CSSProperties> = {
      draft: { background: '#6b7280', color: 'white', padding: '2px 8px', borderRadius: '4px', fontSize: '12px' },
      open: { background: '#10b981', color: 'white', padding: '2px 8px', borderRadius: '4px', fontSize: '12px' },
      closed: { background: '#ef4444', color: 'white', padding: '2px 8px', borderRadius: '4px', fontSize: '12px' },
    }
    return <span style={styles[status] || styles.draft}>{status}</span>
  }

  const handlePollClick = (poll: DevicePollSummary) => {
    if (poll.role === 'admin') {
      navigate(`/admin/${poll.poll_id}`)
    } else if (poll.share_slug) {
      if (poll.status === 'closed') {
        navigate(`/poll/${poll.share_slug}/results`)
      } else {
        navigate(`/poll/${poll.share_slug}`)
      }
    }
  }

  if (loading) {
    return (
      <Container>
        <LoadingSpinner message="Loading your polls..." />
      </Container>
    )
  }

  if (error) {
    return (
      <Container>
        <ErrorMessage message={error} />
        <Button onClick={() => navigate('/')}>Go Home</Button>
      </Container>
    )
  }

  return (
    <Container>
      <main>
        <h1>My Polls</h1>

        <Button onClick={() => navigate('/')} fullWidth={false}>
          &larr; Back to Home
        </Button>

        {polls.length === 0 ? (
          <Card>
            <p>You haven't created or voted on any polls yet.</p>
            <Button onClick={() => navigate('/create')}>Create Your First Poll</Button>
          </Card>
        ) : (
          <>
            {adminPolls.length > 0 && (
              <Card>
                <h2>Polls You Created</h2>
                <ul style={{ listStyle: 'none', padding: 0 }}>
                  {adminPolls.map(poll => (
                    <li
                      key={poll.poll_id}
                      onClick={() => handlePollClick(poll)}
                      style={{
                        padding: '12px',
                        borderBottom: '1px solid #e5e7eb',
                        cursor: 'pointer',
                      }}
                    >
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <strong>{poll.title}</strong>
                        {getStatusBadge(poll.status)}
                      </div>
                      <div style={{ fontSize: '14px', color: '#6b7280', marginTop: '4px' }}>
                        {poll.ballot_count} vote{poll.ballot_count !== 1 ? 's' : ''}
                      </div>
                    </li>
                  ))}
                </ul>
              </Card>
            )}

            {voterPolls.length > 0 && (
              <Card>
                <h2>Polls You Voted On</h2>
                <ul style={{ listStyle: 'none', padding: 0 }}>
                  {voterPolls.map(poll => (
                    <li
                      key={poll.poll_id}
                      onClick={() => handlePollClick(poll)}
                      style={{
                        padding: '12px',
                        borderBottom: '1px solid #e5e7eb',
                        cursor: 'pointer',
                      }}
                    >
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <strong>{poll.title}</strong>
                        {getStatusBadge(poll.status)}
                      </div>
                      <div style={{ fontSize: '14px', color: '#6b7280', marginTop: '4px' }}>
                        Voted as: {poll.username || 'Anonymous'}
                      </div>
                    </li>
                  ))}
                </ul>
              </Card>
            )}
          </>
        )}
      </main>
    </Container>
  )
}
