import type { SliderProps } from '../../types'
import './Slider.css'

const getSemanticLabel = (value: number): string => {
  if (value <= 0.1) return 'Strongly dislike'
  if (value <= 0.3) return 'Dislike'
  if (value <= 0.7) return 'Neutral'
  if (value <= 0.9) return 'Like'
  return 'Strongly like'
}

export const Slider = ({
  label,
  value,
  onChange,
  disabled = false
}: SliderProps) => {
  const sliderId = `slider-${label.toLowerCase().replace(/\s+/g, '-')}`
  const semanticValue = getSemanticLabel(value)
  
  return (
    <div className="slider-group">
      <label htmlFor={sliderId} className="slider-label">
        {label}
      </label>
      <div className="slider-container">
        <input
          id={sliderId}
          type="range"
          min="0"
          max="1"
          step="0.01"
          value={value}
          onChange={(e) => onChange(parseFloat(e.target.value))}
          disabled={disabled}
          className="slider"
          aria-describedby={`${sliderId}-value`}
          aria-valuetext={semanticValue}
        />
        <div 
          id={`${sliderId}-value`} 
          className="slider-value"
          aria-live="polite"
        >
          {semanticValue}
        </div>
      </div>
    </div>
  )
}