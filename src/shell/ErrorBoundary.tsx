import { Component, ErrorInfo, ReactNode } from 'react';

interface Props {
  children: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
  errorInfo: ErrorInfo | null;
}

export class ErrorBoundary extends Component<Props, State> {
  public override state: State = {
    hasError: false,
    error: null,
    errorInfo: null,
  };

  public static getDerivedStateFromError(error: Error): Partial<State> {
    return { hasError: true, error };
  }

  public override componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    this.setState({ errorInfo });
    console.error("AgentVerse Render Error caught by Boundary:", {
      message: error.message,
      stack: error.stack,
      componentStack: errorInfo.componentStack,
    });
  }

  private handleReload = () => {
    window.location.reload();
  };

  private handleReport = () => {
    console.log("Diagnostic report submitted successfully:", {
      error: this.state.error,
      errorInfo: this.state.errorInfo,
    });
    alert("Diagnostic report has been generated and logged to the console.");
  };

  public override render() {
    if (this.state.hasError) {
      return (
        <div className="sentinel-error-container" id="error-boundary-fallback">
          <div className="sentinel-card glow-red animate-pulse">
            <div className="sentinel-card-header">
              <span className="sentinel-dot bg-red blink"></span>
              <span className="sentinel-tag text-red">CRITICAL CORE EXCEPTION</span>
            </div>
            <div className="sentinel-card-body">
              <h2 className="sentinel-title text-red">RENDER FAILURE DETECTED</h2>
              <p className="sentinel-desc">
                A fatal error occurred during page rendering. The active interface has been isolated to protect session memory.
              </p>
              <div className="sentinel-terminal-box error-log">
                <span className="text-red"> &gt; EXCEPTION: {this.state.error?.name || 'UnknownError'}</span>
                <span className="text-white"> &gt; MESSAGE: {this.state.error?.message || 'No description available'}</span>
                {this.state.errorInfo?.componentStack && (
                  <span className="text-muted text-small stack-trace" style={{ display: 'block', whiteSpace: 'pre-wrap', marginTop: '8px' }}>
                    {this.state.errorInfo.componentStack.trim().split('\n').slice(0, 5).join('\n')}
                  </span>
                )}
              </div>
              <div className="error-actions" style={{ display: 'flex', gap: '12px', marginTop: '20px' }}>
                <button
                  className="sentinel-btn btn-primary"
                  onClick={this.handleReload}
                  id="error-reload-btn"
                >
                  Reload Session
                </button>
                <button
                  className="sentinel-btn btn-secondary"
                  onClick={this.handleReport}
                  id="error-report-btn"
                >
                  Report Diagnostic
                </button>
              </div>
            </div>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
export default ErrorBoundary;
