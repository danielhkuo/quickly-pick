import { useState, useCallback } from 'react'

type ValidationRule<T> = (value: T) => string | null
type ValidationRules<T> = {
  [K in keyof T]?: ValidationRule<T[K]>[]
}

interface UseFormValidationReturn<T> {
  errors: Partial<Record<keyof T, string>>
  validate: (data: T) => boolean
  validateField: (field: keyof T, value: T[keyof T]) => boolean
  clearErrors: () => void
  clearFieldError: (field: keyof T) => void
  setFieldError: (field: keyof T, error: string) => void
}

export function useFormValidation<T extends Record<string, unknown>>(
  rules: ValidationRules<T>
): UseFormValidationReturn<T> {
  const [errors, setErrors] = useState<Partial<Record<keyof T, string>>>({})

  const validateField = useCallback((field: keyof T, value: T[keyof T]): boolean => {
    const fieldRules = rules[field]
    if (!fieldRules) return true

    for (const rule of fieldRules) {
      const error = rule(value)
      if (error) {
        setErrors(prev => ({ ...prev, [field]: error }))
        return false
      }
    }

    setErrors(prev => {
      const newErrors = { ...prev }
      delete newErrors[field]
      return newErrors
    })
    return true
  }, [rules])

  const validate = useCallback((data: T): boolean => {
    const newErrors: Partial<Record<keyof T, string>> = {}
    let isValid = true

    for (const field in rules) {
      const fieldRules = rules[field]
      if (!fieldRules) continue

      const value = data[field]
      for (const rule of fieldRules) {
        const error = rule(value)
        if (error) {
          newErrors[field] = error
          isValid = false
          break
        }
      }
    }

    setErrors(newErrors)
    return isValid
  }, [rules])

  const clearErrors = useCallback(() => {
    setErrors({})
  }, [])

  const clearFieldError = useCallback((field: keyof T) => {
    setErrors(prev => {
      const newErrors = { ...prev }
      delete newErrors[field]
      return newErrors
    })
  }, [])

  const setFieldError = useCallback((field: keyof T, error: string) => {
    setErrors(prev => ({ ...prev, [field]: error }))
  }, [])

  return {
    errors,
    validate,
    validateField,
    clearErrors,
    clearFieldError,
    setFieldError
  }
}

// Common validation rules
export const validationRules = {
  required: <T>(message = 'This field is required') => (value: T): string | null => {
    if (value === null || value === undefined || value === '') {
      return message
    }
    if (typeof value === 'string' && value.trim() === '') {
      return message
    }
    return null
  },

  minLength: (min: number, message?: string) => (value: string): string | null => {
    if (value && value.length < min) {
      return message || `Must be at least ${min} characters`
    }
    return null
  },

  maxLength: (max: number, message?: string) => (value: string): string | null => {
    if (value && value.length > max) {
      return message || `Must be no more than ${max} characters`
    }
    return null
  },

  email: (message = 'Please enter a valid email address') => (value: string): string | null => {
    if (value && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)) {
      return message
    }
    return null
  },

  minArrayLength: (min: number, message?: string) => (value: unknown[]): string | null => {
    if (value && value.length < min) {
      return message || `Must have at least ${min} items`
    }
    return null
  }
}