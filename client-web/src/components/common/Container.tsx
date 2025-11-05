import type { ReactNode } from 'react'
import './Container.css'

interface ContainerProps {
  children: ReactNode
  className?: string
}

export const Container = ({ children, className = '' }: ContainerProps) => {
  return (
    <div className={`container ${className}`}>
      {children}
    </div>
  )
}