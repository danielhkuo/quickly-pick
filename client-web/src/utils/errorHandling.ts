import { ApiError } from '../types'

/**
 * Formats an error for user display
 */
export function formatErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.status === 0) {
      return 'Unable to connect to the server. Please check your internet connection.'
    } else if (error.status >= 500) {
      return 'Server error. Please try again later.'
    } else if (error.status === 404) {
      return 'The requested resource was not found.'
    } else if (error.status === 403) {
      return 'Access denied. Please check your permissions.'
    } else if (error.status === 401) {
      return 'Authentication required. Please log in again.'
    } else if (error.status === 400) {
      return error.message || 'Invalid request. Please check your input.'
    } else {
      return error.message || 'An error occurred while processing your request.'
    }
  } else if (error instanceof Error) {
    return error.message
  } else if (typeof error === 'string') {
    return error
  } else {
    return 'An unexpected error occurred.'
  }
}

/**
 * Determines if an error is a network-related error
 */
export function isNetworkError(error: unknown): boolean {
  if (error instanceof ApiError) {
    return error.status === 0
  }
  
  if (error instanceof Error) {
    const message = error.message.toLowerCase()
    return message.includes('network') || 
           message.includes('fetch') || 
           message.includes('connection') ||
           message.includes('timeout')
  }
  
  return false
}

/**
 * Determines if an error is retryable
 */
export function isRetryableError(error: unknown): boolean {
  if (error instanceof ApiError) {
    // Retry on server errors and network errors
    return error.status >= 500 || error.status === 0
  }
  
  return isNetworkError(error)
}

/**
 * Gets a user-friendly retry message based on the error
 */
export function getRetryMessage(error: unknown): string {
  if (isNetworkError(error)) {
    return 'Check your connection and try again'
  } else if (error instanceof ApiError && error.status >= 500) {
    return 'Server issue - try again in a moment'
  } else {
    return 'Try again'
  }
}

/**
 * Logs errors for debugging while being safe for production
 */
export function logError(error: unknown, context?: string): void {
  const isDevelopment = import.meta.env.DEV
  
  if (isDevelopment) {
    console.error(`Error${context ? ` in ${context}` : ''}:`, error)
  } else {
    // In production, log minimal information
    const errorInfo = {
      message: formatErrorMessage(error),
      context,
      timestamp: new Date().toISOString()
    }
    console.error('Application error:', errorInfo)
  }
}