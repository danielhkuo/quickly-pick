import { useState, useCallback } from 'react'
import { formatErrorMessage, logError } from '../utils/errorHandling'

interface AsyncState<T> {
  loading: boolean
  error: string | null
  data: T | null
}

interface UseAsyncOperationReturn<T> {
  state: AsyncState<T>
  execute: (operation: () => Promise<T>) => Promise<T | null>
  reset: () => void
  setData: (data: T | null) => void
  setError: (error: string | null) => void
}

export function useAsyncOperation<T = unknown>(
  initialData: T | null = null
): UseAsyncOperationReturn<T> {
  const [state, setState] = useState<AsyncState<T>>({
    loading: false,
    error: null,
    data: initialData
  })

  const execute = useCallback(async (operation: () => Promise<T>): Promise<T | null> => {
    setState(prev => ({ ...prev, loading: true, error: null }))
    
    try {
      const result = await operation()
      setState({ loading: false, error: null, data: result })
      return result
    } catch (error) {
      logError(error, 'useAsyncOperation')
      const errorMessage = formatErrorMessage(error)
      setState({ loading: false, error: errorMessage, data: null })
      return null
    }
  }, [])

  const reset = useCallback(() => {
    setState({ loading: false, error: null, data: initialData })
  }, [initialData])

  const setData = useCallback((data: T | null) => {
    setState(prev => ({ ...prev, data }))
  }, [])

  const setError = useCallback((error: string | null) => {
    setState(prev => ({ ...prev, error }))
  }, [])

  return { state, execute, reset, setData, setError }
}