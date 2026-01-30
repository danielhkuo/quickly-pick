// API Response Types
export interface Poll {
  id: string
  title: string
  description: string
  creator_name: string
  status: 'draft' | 'open' | 'closed'
  share_slug?: string
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
  option_id: string
  label: string
  rank: number
  median: number
  p10: number
  p90: number
  mean: number
  neg_share: number
  veto: boolean
}

// API Request Types
export interface CreatePollRequest {
  title: string
  description: string
  creator_name: string
}

export interface AddOptionRequest {
  label: string
}



export interface ClaimUsernameRequest {
  username: string
}

export interface SubmitBallotRequest {
  scores: Record<string, number> // option_id -> rating (0.0 to 1.0)
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
  required?: boolean
  disabled?: boolean
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

// Device Types
export interface DeviceInfo {
  id: string
  platform: 'ios' | 'macos' | 'android' | 'web'
  created_at: string
  last_seen_at: string
}

export interface DevicePollSummary {
  poll_id: string
  title: string
  status: 'draft' | 'open' | 'closed'
  share_slug?: string
  role: 'voter' | 'admin'
  username?: string
  ballot_count: number
  linked_at: string
}