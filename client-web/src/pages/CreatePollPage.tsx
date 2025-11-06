import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Container, Card, Button, Input, ErrorMessage } from '../components/common'
import { useAsyncOperation } from '../hooks/useAsyncOperation'
import { useFormValidation, validationRules } from '../hooks/useFormValidation'
import { apiClient } from '../api/client'
import type { CreatePollRequest, AddOptionRequest } from '../types'
import './CreatePollPage.css'

interface PollData {
  title: string
  description: string
  creator_name: string
}

interface PollOption {
  id: string
  label: string
  position: number
}

type WizardStep = 1 | 2 | 3

export const CreatePollPage = () => {
  const navigate = useNavigate()
  const [currentStep, setCurrentStep] = useState<WizardStep>(1)
  
  // Async operation management
  const publishOperation = useAsyncOperation<{ poll_id: string; admin_key: string }>()

  // Step 1: Poll Details
  const [pollData, setPollData] = useState<PollData>({
    title: '',
    description: '',
    creator_name: ''
  })

  // Form validation for poll details
  const pollValidation = useFormValidation<PollData>({
    title: [validationRules.required('Poll title is required')],
    creator_name: [validationRules.required('Creator name is required')],
    description: [] // Optional field
  })

  // Step 2: Options Management
  const [options, setOptions] = useState<PollOption[]>([
    { id: '1', label: '', position: 1 },
    { id: '2', label: '', position: 2 }
  ])
  const [optionErrors, setOptionErrors] = useState<Record<string, string>>({})

  // Step 3: Review and Publish

  const validateStep1 = (): boolean => {
    return pollValidation.validate(pollData)
  }

  const validateStep2 = (): boolean => {
    const errors: Record<string, string> = {}
    const validOptions = options.filter(opt => opt.label.trim())
    
    if (validOptions.length < 2) {
      errors.general = 'At least 2 options are required'
    }

    options.forEach(option => {
      if (option.label.trim() && validOptions.filter(opt => opt.label.trim() === option.label.trim()).length > 1) {
        errors[option.id] = 'Option labels must be unique'
      }
    })

    setOptionErrors(errors)
    return Object.keys(errors).length === 0
  }

  const handleStep1Next = () => {
    if (validateStep1()) {
      setCurrentStep(2)
    }
  }

  const handleStep2Next = () => {
    if (validateStep2()) {
      setCurrentStep(3)
    }
  }

  const handleStep2Back = () => {
    setCurrentStep(1)
  }

  const handleStep3Back = () => {
    setCurrentStep(2)
  }

  const addOption = () => {
    const newId = (Math.max(...options.map(opt => parseInt(opt.id))) + 1).toString()
    setOptions([...options, {
      id: newId,
      label: '',
      position: options.length + 1
    }])
  }

  const removeOption = (id: string) => {
    if (options.length > 2) {
      setOptions(options.filter(opt => opt.id !== id))
    }
  }

  const updateOption = (id: string, label: string) => {
    setOptions(options.map(opt => 
      opt.id === id ? { ...opt, label } : opt
    ))
  }

  const handlePublish = async () => {
    const result = await publishOperation.execute(async () => {
      // Step 1: Create the poll
      const createRequest: CreatePollRequest = {
        title: pollData.title.trim(),
        description: pollData.description.trim(),
        creator_name: pollData.creator_name.trim()
      }

      const createResponse = await apiClient.createPoll(createRequest)

      // Step 2: Add all options
      const validOptions = options.filter(opt => opt.label.trim())
      for (const option of validOptions) {
        const addOptionRequest: AddOptionRequest = {
          label: option.label.trim()
        }
        await apiClient.addOption(createResponse.poll_id, addOptionRequest, createResponse.admin_key)
      }

      // Step 3: Publish the poll
      const publishResponse = await apiClient.publishPoll(createResponse.poll_id, createResponse.admin_key)
      
      // Ensure the poll is published before returning
      console.log('Poll published successfully:', publishResponse)

      return createResponse
    })

    if (result) {
      // Store admin key in localStorage
      localStorage.setItem(`admin_key_${result.poll_id}`, result.admin_key)
      // Navigate to admin interface
      navigate(`/admin/${result.poll_id}`)
    }
  }

  const renderProgressIndicator = () => (
    <div className="progress-indicator">
      <div className={`progress-step ${currentStep >= 1 ? 'active' : ''}`}>
        <span className="step-number">1</span>
        <span className="step-label">Details</span>
      </div>
      <div className={`progress-step ${currentStep >= 2 ? 'active' : ''}`}>
        <span className="step-number">2</span>
        <span className="step-label">Options</span>
      </div>
      <div className={`progress-step ${currentStep >= 3 ? 'active' : ''}`}>
        <span className="step-number">3</span>
        <span className="step-label">Review</span>
      </div>
    </div>
  )

  const renderStep1 = () => (
    <Card>
      <h2>Poll Details</h2>
      <p>Enter the basic information for your poll.</p>
      
      <Input
        label="Poll Title"
        value={pollData.title}
        onChange={(value) => {
          setPollData({ ...pollData, title: value })
          pollValidation.clearFieldError('title')
        }}
        error={pollValidation.errors.title}
        placeholder="What would you like to ask?"
        required
      />

      <Input
        label="Description (Optional)"
        value={pollData.description}
        onChange={(value) => setPollData({ ...pollData, description: value })}
        placeholder="Provide additional context or instructions"
      />

      <Input
        label="Your Name"
        value={pollData.creator_name}
        onChange={(value) => {
          setPollData({ ...pollData, creator_name: value })
          pollValidation.clearFieldError('creator_name')
        }}
        error={pollValidation.errors.creator_name}
        placeholder="How should voters know who created this poll?"
        required
      />

      <div className="step-actions">
        <Button onClick={handleStep1Next} fullWidth>
          Next: Add Options
        </Button>
      </div>
    </Card>
  )

  const renderStep2 = () => (
    <Card>
      <h2>Poll Options</h2>
      <p>Add the options that voters will choose between. You need at least 2 options.</p>
      
      {optionErrors.general && (
        <div className="error-message" role="alert">
          {optionErrors.general}
        </div>
      )}

      <div className="options-list">
        {options.map((option, index) => (
          <div key={option.id} className="option-item">
            <Input
              label={`Option ${index + 1}`}
              value={option.label}
              onChange={(value) => updateOption(option.id, value)}
              error={optionErrors[option.id]}
              placeholder={`Enter option ${index + 1}`}
            />
            {options.length > 2 && (
              <Button 
                onClick={() => removeOption(option.id)}
                type="button"
              >
                Remove
              </Button>
            )}
          </div>
        ))}
      </div>

      <Button onClick={addOption} type="button">
        Add Another Option
      </Button>

      <div className="step-actions">
        <Button onClick={handleStep2Back}>
          Back: Poll Details
        </Button>
        <Button onClick={handleStep2Next}>
          Next: Review & Publish
        </Button>
      </div>
    </Card>
  )

  const renderStep3 = () => (
    <Card>
      <h2>Review & Publish</h2>
      <p>Review your poll details before publishing. Once published, voters can start participating.</p>
      
      <div className="poll-summary">
        <h3>Poll Details</h3>
        <div className="summary-item">
          <strong>Title:</strong> {pollData.title}
        </div>
        {pollData.description && (
          <div className="summary-item">
            <strong>Description:</strong> {pollData.description}
          </div>
        )}
        <div className="summary-item">
          <strong>Created by:</strong> {pollData.creator_name}
        </div>

        <h3>Options ({options.filter(opt => opt.label.trim()).length})</h3>
        <ol className="options-summary">
          {options
            .filter(opt => opt.label.trim())
            .map((option) => (
              <li key={option.id}>{option.label}</li>
            ))}
        </ol>
      </div>

      {publishOperation.state.error && (
        <ErrorMessage 
          message={publishOperation.state.error}
          onRetry={handlePublish}
          retryLabel="Try Publishing Again"
        />
      )}

      <div className="step-actions">
        <Button onClick={handleStep3Back} disabled={publishOperation.state.loading}>
          Back: Edit Options
        </Button>
        <Button 
          onClick={handlePublish} 
          disabled={publishOperation.state.loading}
          type="submit"
        >
          {publishOperation.state.loading ? 'Publishing...' : 'Publish Poll'}
        </Button>
      </div>
    </Card>
  )

  return (
    <Container>
      <div className="create-poll-page">
        <h1>Create New Poll</h1>
        
        {renderProgressIndicator()}
        
        {currentStep === 1 && renderStep1()}
        {currentStep === 2 && renderStep2()}
        {currentStep === 3 && renderStep3()}
      </div>
    </Container>
  )
}