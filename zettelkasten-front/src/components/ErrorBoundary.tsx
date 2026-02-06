import React, { Component, ErrorInfo, ReactNode } from 'react';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
}

interface State {
  hasError: boolean;
  error?: Error;
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(error: Error): State {
    // Update state so the next render will show the fallback UI
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    // Log the error for debugging purposes
    console.error('ErrorBoundary caught an error:', error, errorInfo);

    // Call optional error handler
    if (this.props.onError) {
      this.props.onError(error, errorInfo);
    }
  }

  render() {
    if (this.state.hasError) {
      // Render custom fallback UI if provided, otherwise default
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return (
        <div className="p-4 m-4 border border-red-300 rounded bg-red-50">
          <h2 className="text-lg font-semibold text-red-800 mb-2">
            Something went wrong
          </h2>
          <p className="text-red-600 mb-3">
            We encountered an error while displaying this content. Please try refreshing the page.
          </p>
          <button
            className="px-4 py-3 min-h-[44px] bg-red-600 text-white rounded hover:bg-red-700 transition-colors"
            onClick={() => window.location.reload()}
          >
            Refresh Page
          </button>
          {process.env.NODE_ENV === 'development' && this.state.error && (
            <details className="mt-4">
              <summary className="cursor-pointer text-red-700 font-medium">
                Error Details (Development)
              </summary>
              <pre className="mt-2 p-2 bg-red-100 rounded text-xs overflow-auto text-red-800">
                {this.state.error.toString()}
              </pre>
            </details>
          )}
        </div>
      );
    }

    return this.props.children;
  }
}

// Specialized error boundary for pinned card failures
interface PinErrorBoundaryProps {
  children: ReactNode;
  onPinError?: () => void;
}

export const PinErrorBoundary: React.FC<PinErrorBoundaryProps> = ({
  children,
  onPinError
}) => {
  const handleError = (error: Error, errorInfo: ErrorInfo) => {
    console.error('Pinned card error:', error, errorInfo);
    if (onPinError) {
      onPinError();
    }
  };

  const fallback = (
    <div className="p-4 bg-red-50 border border-red-200 rounded">
      <h3 className="text-red-800 font-medium mb-2">
        Failed to load pinned card
      </h3>
      <p className="text-red-600 text-sm mb-3">
        There was an error loading the pinned card. The application will continue to work normally.
      </p>
      <button
        className="text-sm px-3 py-1 bg-red-600 text-white rounded hover:bg-red-700 transition-colors"
        onClick={onPinError}
      >
        Close Pin View
      </button>
    </div>
  );

  return (
    <ErrorBoundary fallback={fallback} onError={handleError}>
      {children}
    </ErrorBoundary>
  );
};