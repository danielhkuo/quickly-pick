import { API_BASE_URL } from './config'
import { ApiError } from '../types'
import type {
  CreatePollRequest,
  CreatePollResponse,
  AddOptionRequest,
  PublishPollRequest,
  ClosePollRequest,
  ClaimUsernameRequest,
  ClaimUsernameResponse,
  SubmitBallotRequest,
  GetPollResponse,
  GetResultsResponse,
  Option
} from '../types'

/**
 * Base API client with error handling
 */
class ApiClient {
  private baseUrl: string

  constructor(baseUrl: string = API_BASE_URL) {
    this.baseUrl = baseUrl
  }

  /**
   * Generic fetch wrapper with error handling
   */
  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`
    
    const config: RequestInit = {
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
      ...options,
    }

    try {
      const response = await fetch(url, config)
      
      if (!response.ok) {
        const errorText = await response.text()
        throw new ApiError(response.status, errorText || `HTTP ${response.status}`)
      }

      // Handle empty responses
      const contentType = response.headers.get('content-type')
      if (contentType && contentType.includes('application/json')) {
        return await response.json()
      } else {
        return {} as T
      }
    } catch (error) {
      if (error instanceof ApiError) {
        throw error
      }
      // Network or other errors
      throw new ApiError(0, error instanceof Error ? error.message : 'Network error')
    }
  }

  /**
   * GET request helper
   */
  private get<T>(endpoint: string, headers?: Record<string, string>): Promise<T> {
    return this.request<T>(endpoint, { method: 'GET', headers })
  }

  /**
   * POST request helper
   */
  private post<T>(
    endpoint: string,
    data?: unknown,
    headers?: Record<string, string>
  ): Promise<T> {
    return this.request<T>(endpoint, {
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
      headers,
    })
  }

  /**
   * PUT request helper
   */
  private put<T>(
    endpoint: string,
    data?: unknown,
    headers?: Record<string, string>
  ): Promise<T> {
    return this.request<T>(endpoint, {
      method: 'PUT',
      body: data ? JSON.stringify(data) : undefined,
      headers,
    })
  }

  // Poll Management Functions

  /**
   * Create a new poll
   */
  async createPoll(pollData: CreatePollRequest): Promise<CreatePollResponse> {
    return this.post<CreatePollResponse>('/api/polls', pollData)
  }

  /**
   * Add an option to a poll
   */
  async addOption(pollId: string, optionData: AddOptionRequest, adminKey: string): Promise<Option> {
    return this.post<Option>(`/api/polls/${pollId}/options`, optionData, {
      'Authorization': `Bearer ${adminKey}`
    })
  }

  /**
   * Publish a poll (make it available for voting)
   */
  async publishPoll(pollId: string, publishData: PublishPollRequest): Promise<void> {
    return this.put<void>(`/api/polls/${pollId}/publish`, publishData)
  }

  /**
   * Close a poll (stop accepting votes)
   */
  async closePoll(pollId: string, closeData: ClosePollRequest): Promise<void> {
    return this.put<void>(`/api/polls/${pollId}/close`, closeData)
  }

  /**
   * Get poll status and metadata (admin view)
   */
  async getPollAdmin(pollId: string, adminKey: string): Promise<GetPollResponse> {
    return this.get<GetPollResponse>(`/api/admin/polls/${pollId}`, {
      'Authorization': `Bearer ${adminKey}`
    })
  }

  // Voting Functions

  /**
   * Get poll details for voting
   */
  async getPoll(slug: string): Promise<GetPollResponse> {
    return this.get<GetPollResponse>(`/api/polls/${slug}`)
  }

  /**
   * Claim a username for voting
   */
  async claimUsername(slug: string, usernameData: ClaimUsernameRequest): Promise<ClaimUsernameResponse> {
    return this.post<ClaimUsernameResponse>(`/api/polls/${slug}/claim-username`, usernameData)
  }

  /**
   * Submit a ballot with ratings
   */
  async submitBallot(slug: string, ballotData: SubmitBallotRequest): Promise<void> {
    return this.post<void>(`/api/polls/${slug}/vote`, ballotData)
  }

  // Results Functions

  /**
   * Get poll results
   */
  async getResults(slug: string): Promise<GetResultsResponse> {
    return this.get<GetResultsResponse>(`/api/polls/${slug}/results`)
  }

  /**
   * Get ballot count for a poll
   */
  async getBallotCount(slug: string): Promise<{ ballot_count: number }> {
    return this.get<{ ballot_count: number }>(`/api/polls/${slug}/ballot-count`)
  }
}

// Export singleton instance
export const apiClient = new ApiClient()

// Export class for testing
export { ApiClient }