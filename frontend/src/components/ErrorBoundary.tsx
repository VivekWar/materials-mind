import React, { Component, ErrorInfo, ReactNode } from 'react'
import { AlertTriangle, RefreshCcw } from 'lucide-react'
import { Button } from './ui/button'

interface Props {
  children?: ReactNode
}

interface State {
  hasError: boolean
  error: Error | null
}

export class ErrorBoundary extends Component<Props, State> {
  public state: State = {
    hasError: false,
    error: null,
  }

  public static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  public componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('Uncaught component error:', error, errorInfo)
  }

  public render() {
    if (this.state.hasError) {
      return (
        <div className="flex flex-col items-center justify-center min-h-screen bg-background text-foreground p-4 text-center">
          <div className="w-16 h-16 rounded-full bg-red-500/10 flex items-center justify-center mb-4">
            <AlertTriangle className="text-red-500" size={32} />
          </div>
          <h1 className="text-2xl font-bold mb-2">Something went wrong</h1>
          <p className="text-muted-foreground mb-6 max-w-md">
            The application encountered an unexpected rendering error. 
            We apologize for the inconvenience.
          </p>
          {this.state.error && (
            <div className="bg-muted text-left p-4 rounded-md mb-6 w-full max-w-lg overflow-auto">
              <code className="text-xs text-red-400 break-words">
                {this.state.error.toString()}
              </code>
            </div>
          )}
          <Button 
            onClick={() => window.location.reload()}
            className="gap-2"
          >
            <RefreshCcw size={16} />
            Reload Application
          </Button>
        </div>
      )
    }

    return this.props.children
  }
}
