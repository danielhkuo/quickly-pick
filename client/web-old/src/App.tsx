import { BrowserRouter as Router, Routes, Route } from 'react-router-dom'
import { Suspense, lazy } from 'react'
import { ErrorBoundary, LoadingSpinner } from './components/common'

// Lazy load all page components for code splitting
const HomePage = lazy(() => import('./pages/HomePage').then(module => ({ default: module.HomePage })))
const CreatePollPage = lazy(() => import('./pages/CreatePollPage').then(module => ({ default: module.CreatePollPage })))
const AdminPage = lazy(() => import('./pages/AdminPage').then(module => ({ default: module.AdminPage })))
const VotePage = lazy(() => import('./pages/VotePage').then(module => ({ default: module.VotePage })))
const ResultsPage = lazy(() => import('./pages/ResultsPage').then(module => ({ default: module.ResultsPage })))

function App() {
  return (
    <Router>
      <ErrorBoundary>
        <Suspense fallback={<LoadingSpinner message="Loading page..." />}>
          <Routes>
            <Route path="/" element={<HomePage />} />
            <Route path="/create" element={<CreatePollPage />} />
            <Route path="/admin/:pollId" element={<AdminPage />} />
            <Route path="/poll/:slug" element={<VotePage />} />
            <Route path="/poll/:slug/results" element={<ResultsPage />} />
          </Routes>
        </Suspense>
      </ErrorBoundary>
    </Router>
  )
}

export default App
