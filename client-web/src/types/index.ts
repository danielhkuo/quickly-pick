// API Response Types
export interface Poll {
  id: string
  title: string
  description: string
  creator_name: string
  status: 'draft' | 'open' | 'closed'
  slug: string
  created_at: string
  closed_at?: string
}

export interface Option {
  id: string
  poll_id: string
  label: string
  position: number
}

export interface CreatePollResponse {
  poll_id: string
  admin_key: string
}

export interface ClaimUsernameResponse {
  voter_token: string
}

export interface ResultSnapshot {
  poll: Poll
  rankings: OptionRanking[]
  ballot_count: number
}

export interface OptionRanking {
  option: Option
  rank: number
  median: number
  percentile_10: number
  percentile_90: number
  mean: number
  negative_vote_percentage: number
  is_vetoed: boolean
}

// API Request Types
export interface CreatePollRequest {
  title: string
  description: string
  creator_name: string
}

export interface AddOptionRequest {
  label: string
  position: number
}

export interface PublishPollRequest {
  admin_key: string
}

export interface ClosePollRequest {
  admin_key: string
}

export interface ClaimUsernameRequest {
  username: string
}

export interface SubmitBallotRequest {
  voter_token: string
  ratings: Record<string, number> // option_id -> rating (0.0 to 1.0)
}

export interface GetPollResponse {
  poll: Poll
  options: Option[]
}

export interface GetResultsResponse {
  poll: Poll
  rankings: OptionRanking[]
  ballot_count: number
}

// Component Props Types
export interface SliderProps {
  label: string
  value: number        // 0.0 to 1.0
  onChange: (value: number) => void
  disabled?: boolean
}

export interface ButtonProps {
  children: React.ReactNode
  onClick?: () => void
  type?: 'button' | 'submit'
  disabled?: boolean
  fullWidth?: boolean
}

export interface InputProps {
  label: string
  value: string
  onChange: (value: string) => void
  type?: 'text' | 'email'
  error?: string
  placeholder?: string
}

// API Error Type
export class ApiError extends Error {
  public status: number
  
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

// Component State Type
export interface ComponentState<T = unknown> {
  loading: boolean
  error: string | null
  data: T | null
}