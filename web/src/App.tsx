import React from 'react';
import { AppProvider } from '@src/providers/app';
import { AppRoutes } from '@src/routes';

function App() {
  return (
    <AppProvider>
      <AppRoutes />
    </AppProvider>
  );
}

export { App };