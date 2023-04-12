import React from 'react';
import { QueryClientProvider } from 'react-query';
import { ErrorBoundary } from 'react-error-boundary';

import { ToastProvider } from '@src/context/ToastContext';
import { queryClient } from '@src/lib';

import { BrowserRouter } from 'react-router-dom';
type AppProviderProps = {
  children: React.ReactNode;
}

const ErrorFallback = () => {
  return (
    <div
      className="text-red-500 w-screen h-screen flex flex-col justify-center items-center"
      role="alert"
    >
      <h2 className="text-lg font-semibold">Ooops, something went wrong :( </h2>
      {/* <Button className="mt-4" onClick={() => window.location.assign(window.location.origin)}>
        Refresh
      </Button> */}
    </div>
  );
};

const AppProvider = ({ children }: AppProviderProps) => {
  return (
    <ErrorBoundary FallbackComponent={ErrorFallback}>
      <QueryClientProvider client={queryClient}>
        <ToastProvider>
          <BrowserRouter>{children}</BrowserRouter>
        </ToastProvider>
      </QueryClientProvider>
    </ErrorBoundary>
  )
}

export { AppProvider };