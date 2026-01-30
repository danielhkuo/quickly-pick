import { useNavigate } from 'react-router-dom'
import { Container } from '../components/common/Container'
import { Button } from '../components/common/Button'
import { Card } from '../components/common/Card'

export function HomePage() {
  const navigate = useNavigate()

  const handleCreatePoll = () => {
    navigate('/create')
  }

  const handleMyPolls = () => {
    navigate('/my-polls')
  }

  return (
    <Container>
      <main>
        <h1>Balanced Majority Judgment Polls</h1>

        <Card>
          <h2>Create Your Poll</h2>
          <p>
            Gather nuanced feedback from your community using Balanced Majority Judgment.
            Participants rate each option on a scale, providing richer insights than simple voting.
          </p>
          <Button onClick={handleCreatePoll} fullWidth>
            Create New Poll
          </Button>
          <div style={{ marginTop: '12px' }}>
            <Button onClick={handleMyPolls} fullWidth>
              My Polls
            </Button>
          </div>
        </Card>

        <Card>
          <h3>How It Works</h3>
          <ol>
            <li>Create a poll with your options</li>
            <li>Share the link with participants</li>
            <li>Participants rate each option using sliders</li>
            <li>View ranked results based on collective judgment</li>
          </ol>
        </Card>
      </main>
    </Container>
  )
}