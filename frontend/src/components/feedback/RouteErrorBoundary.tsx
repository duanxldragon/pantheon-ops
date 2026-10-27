import React from 'react';
import PageError from './PageError';

interface RouteErrorBoundaryProps {
  children: React.ReactNode;
}

interface RouteErrorBoundaryState {
  error: unknown;
}

/**
 * Catch-all boundary for route-level failures: lazy chunk load errors
 * (e.g. stale hashes after a release) and unexpected render crashes.
 * Keeps a broken route from white-screening the whole app.
 */
class RouteErrorBoundary extends React.Component<RouteErrorBoundaryProps, RouteErrorBoundaryState> {
  state: RouteErrorBoundaryState = { error: null };

  static getDerivedStateFromError(error: unknown): RouteErrorBoundaryState {
    return { error };
  }

  handleRetry = () => {
    globalThis.location.reload();
  };

  render() {
    if (this.state.error) {
      return <PageError onRetry={this.handleRetry} />;
    }
    return this.props.children;
  }
}

export default RouteErrorBoundary;
