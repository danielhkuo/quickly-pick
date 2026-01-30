import { API_BASE_URL } from './config'
import { ApiError } from '../types'
import { getDeviceUUID } from './device'
import type {
  CreatePollRequest,
  CreatePollResponse,
  AddOptionRequest,
  ClaimUsernameRequest,
  ClaimUsernameResponse,
  SubmitBallotRequest,
  GetPollResponse,
  GetResultsResponse,
  DeviceInfo,
  DevicePollSummary
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
        'X-Device-UUID': getDeviceUUID(),
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



  // Poll Management Functions

  /**
   * Create a new poll
   */
  async createPoll(pollData: CreatePollRequest): Promise<CreatePollResponse> {
    return this.post<CreatePollResponse>('/polls', pollData)
  }

  /**
   * Add an option to a poll
   */
  async addOption(pollId: string, optionData: AddOptionRequest, adminKey: string): Promise<{ option_id: string }> {
    return this.post<{ option_id: string }>(`/polls/${pollId}/options`, optionData, {
      'X-Admin-Key': adminKey
    })
  }

  /**
   * Publish a poll (make it available for voting)
   */
  async publishPoll(pollId: string, adminKey: string): Promise<{ share_slug: string; share_url: string }> {
    return this.post<{ share_slug: string; share_url: string }>(`/polls/${pollId}/publish`, {}, {
      'X-Admin-Key': adminKey
    })
  }

  /**
   * Close a poll (stop accepting votes)
   */
  async closePoll(pollId: string, adminKey: string): Promise<void> {
    return this.post<void>(`/polls/${pollId}/close`, {}, {
      'X-Admin-Key': adminKey
    })
  }

  /**
   * Get poll status and metadata (admin view)
   */
  async getPollAdmin(pollId: string, adminKey: string): Promise<GetPollResponse> {
    return this.get<GetPollResponse>(`/polls/${pollId}/admin`, {
      'X-Admin-Key': adminKey
    })
  }

  // Voting Functions

  /**
   * Get poll details for voting
   */
  async getPoll(slug: string): Promise<GetPollResponse> {
    return this.get<GetPollResponse>(`/polls/${slug}`)
  }

  /**
   * Claim a username for voting
   */
  async claimUsername(slug: string, usernameData: ClaimUsernameRequest): Promise<ClaimUsernameResponse> {
    return this.post<ClaimUsernameResponse>(`/polls/${slug}/claim-username`, usernameData)
  }

  /**
   * Submit a ballot with ratings
   * Note: Backend uses /ballots endpoint, not /vote
   */
  async submitBallot(slug: string, ballotData: SubmitBallotRequest, voterToken: string): Promise<void> {
    return this.post<void>(`/polls/${slug}/ballots`, ballotData, {
      'X-Voter-Token': voterToken
    })
  }

  // Results Functions

  /**
   * Get poll results
   */
  async getResults(slug: string): Promise<GetResultsResponse> {
    return this.get<GetResultsResponse>(`/polls/${slug}/results`)
  }

  /**
   * Get ballot count for a poll
   */
  async getBallotCount(slug: string): Promise<{ ballot_count: number }> {
    return this.get<{ ballot_count: number }>(`/polls/${slug}/ballot-count`)
  }

  // Device Functions

  /**
   * Register this device with the backend
   */
  async registerDevice(): Promise<DeviceInfo> {
    return this.post<DeviceInfo>('/devices/register')
  }

  /**
   * Get device info for current device
   */
  async getDeviceInfo(): Promise<DeviceInfo> {
    return this.get<DeviceInfo>('/devices/me')
  }

  /**
   * Get polls associated with this device
   */
  async getMyPolls(): Promise<DevicePollSummary[]> {
    return this.get<DevicePollSummary[]>('/devices/my-polls')
  }
}

// Export singleton instance
export const apiClient = new ApiClient()

// Export class for testing
export { ApiClient }