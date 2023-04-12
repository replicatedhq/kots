import React from 'react';
import { QueryClientProvider } from 'react-query';

import { ToastProvider } from '@src/context/ToastContext';
import { queryClient } from '@src/lib';

import { BrowserRouter as Router } from 'react-router-dom-6';
type AppProviderProps = {
  children: React.ReactNode;
}

const AppProvider = ({ children }: AppProviderProps) => {
  return (
    <QueryClientProvider client={queryClient}>
      <ToastProvider>
        <Router>{children}</Router>
      </ToastProvider>
    </QueryClientProvider>
  )
}

export { AppProvider };