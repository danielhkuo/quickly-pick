import { BrowserRouter as Router, Routes, Route } from 'react-router-dom'
import { Suspense } from 'react'

// Placeholder components for now - will be implemented in later tasks
const HomePage = () => <div>Home Page - Coming Soon</div>
const CreatePollPage = () => <div>Create Poll Page - Coming Soon</div>
const AdminPage = () => <div>Admin Page - Coming Soon</div>
const VotePage = () => <div>Vote Page - Coming Soon</div>
const ResultsPage = () => <div>Results Page - Coming Soon</div>

function App() {
  return (
    <Router>
      <Suspense fallback={<div>Loading...</div>}>
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/create" element={<CreatePollPage />} />
          <Route path="/admin/:pollId" element={<AdminPage />} />
          <Route path="/poll/:slug" element={<VotePage />} />
          <Route path="/poll/:slug/results" element={<ResultsPage />} />
        </Routes>
      </Suspense>
    </Router>
  )
}

export default App
