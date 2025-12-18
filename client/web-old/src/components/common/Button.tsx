import type { ButtonProps } from '../../types'
import './Button.css'

export const Button = ({ 
  children, 
  onClick, 
  type = 'button', 
  disabled = false, 
  fullWidth = false 
}: ButtonProps) => {
  return (
    <button
      type={type}
      onClick={onClick}
      disabled={disabled}
      className={`button ${fullWidth ? 'button--full-width' : ''}`}
    >
      {children}
    </button>
  )
}