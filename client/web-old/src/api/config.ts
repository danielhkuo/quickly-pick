// API Configuration
export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:3318'

// API endpoints
export const API_ENDPOINTS = {
  polls: '/api/polls',
  vote: '/api/vote',
  results: '/api/results',
  admin: '/api/admin'
} as const