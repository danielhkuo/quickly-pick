import { apiClient } from './client'
import type {
  CreatePollRequest,
  CreatePollResponse,
  AddOptionRequest,
  GetPollResponse,
  GetResultsResponse,
  Option
} from '../types'

/**
 * Local storage keys for poll-related data
 */
export const STORAGE_KEYS = {
  adminKey: (pollId: string) => `admin_key_${pollId}`,
  voterToken: (slug: string) => `voter_token_${slug}`,
  username: (slug: string) => `username_${slug}`,
} as const

/**
 * Local storage utilities
 */
export const storage = {
  /**
   * Store admin key for a poll
   */
  setAdminKey(pollId: string, adminKey: string): void {
    localStorage.setItem(STORAGE_KEYS.adminKey(pollId), adminKey)
  },

  /**
   * Get admin key for a poll
   */
  getAdminKey(pollId: string): string | null {
    return localStorage.getItem(STORAGE_KEYS.adminKey(pollId))
  },

  /**
   * Store voter token for a poll
   */
  setVoterToken(slug: string, voterToken: string): void {
    localStorage.setItem(STORAGE_KEYS.voterToken(slug), voterToken)
  },

  /**
   * Get voter token for a poll
   */
  getVoterToken(slug: string): string | null {
    return localStorage.getItem(STORAGE_KEYS.voterToken(slug))
  },

  /**
   * Store username for a poll
   */
  setUsername(slug: string, username: string): void {
    localStorage.setItem(STORAGE_KEYS.username(slug), username)
  },

  /**
   * Get username for a poll
   */
  getUsername(slug: string): string | null {
    return localStorage.getItem(STORAGE_KEYS.username(slug))
  },

  /**
   * Clear all data for a poll (admin)
   */
  clearPollData(pollId: string): void {
    localStorage.removeItem(STORAGE_KEYS.adminKey(pollId))
  },

  /**
   * Clear all data for a voter session
   */
  clearVoterData(slug: string): void {
    localStorage.removeItem(STORAGE_KEYS.voterToken(slug))
    localStorage.removeItem(STORAGE_KEYS.username(slug))
  }
}

/**
 * High-level poll management functions
 */
export const pollApi = {
  /**
   * Create a poll and store admin credentials
   */
  async createPoll(pollData: CreatePollRequest): Promise<CreatePollResponse> {
    const response = await apiClient.createPoll(pollData)
    storage.setAdminKey(response.poll_id, response.admin_key)
    return response
  },

  /**
   * Add option to poll using stored admin key
   */
  async addOption(pollId: string, optionData: AddOptionRequest): Promise<Option> {
    const adminKey = storage.getAdminKey(pollId)
    if (!adminKey) {
      throw new Error('Admin key not found for this poll')
    }
    return apiClient.addOption(pollId, optionData, adminKey)
  },

  /**
   * Publish poll using stored admin key
   */
  async publishPoll(pollId: string): Promise<void> {
    const adminKey = storage.getAdminKey(pollId)
    if (!adminKey) {
      throw new Error('Admin key not found for this poll')
    }
    return apiClient.publishPoll(pollId, { admin_key: adminKey })
  },

  /**
   * Close poll using stored admin key
   */
  async closePoll(pollId: string): Promise<void> {
    const adminKey = storage.getAdminKey(pollId)
    if (!adminKey) {
      throw new Error('Admin key not found for this poll')
    }
    return apiClient.closePoll(pollId, { admin_key: adminKey })
  },

  /**
   * Get poll admin view using stored admin key
   */
  async getPollAdmin(pollId: string): Promise<GetPollResponse> {
    const adminKey = storage.getAdminKey(pollId)
    if (!adminKey) {
      throw new Error('Admin key not found for this poll')
    }
    return apiClient.getPollAdmin(pollId, adminKey)
  }
}

/**
 * High-level voting functions
 */
export const votingApi = {
  /**
   * Get poll for voting
   */
  async getPoll(slug: string): Promise<GetPollResponse> {
    return apiClient.getPoll(slug)
  },

  /**
   * Claim username and store credentials
   */
  async claimUsername(slug: string, username: string): Promise<void> {
    const response = await apiClient.claimUsername(slug, { username })
    storage.setVoterToken(slug, response.voter_token)
    storage.setUsername(slug, username)
  },

  /**
   * Submit ballot using stored voter token
   */
  async submitBallot(slug: string, ratings: Record<string, number>): Promise<void> {
    const voterToken = storage.getVoterToken(slug)
    if (!voterToken) {
      throw new Error('Voter token not found. Please claim a username first.')
    }
    return apiClient.submitBallot(slug, { voter_token: voterToken, ratings })
  },

  /**
   * Check if user has already voted (has voter token)
   */
  hasVoted(slug: string): boolean {
    return storage.getVoterToken(slug) !== null
  },

  /**
   * Get claimed username for poll
   */
  getClaimedUsername(slug: string): string | null {
    return storage.getUsername(slug)
  }
}

/**
 * High-level results functions
 */
export const resultsApi = {
  /**
   * Get poll results
   */
  async getResults(slug: string): Promise<GetResultsResponse> {
    return apiClient.getResults(slug)
  },

  /**
   * Get ballot count
   */
  async getBallotCount(slug: string): Promise<number> {
    const response = await apiClient.getBallotCount(slug)
    return response.ballot_count
  }
}