interface InlineErrorProps {
  message: string
  id?: string
}

export function InlineError({ message, id }: InlineErrorProps) {
  return (
    <div 
      id={id}
      style={{ 
        color: '#d32f2f',
        fontSize: '14px',
        marginTop: '4px',
        display: 'flex',
        alignItems: 'center',
        gap: '4px'
      }}
      role="alert"
      aria-live="polite"
    >
      <span aria-hidden="true">⚠</span>
      {message}
    </div>
  )
}