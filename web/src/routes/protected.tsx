import React from 'react';
import { Dashboard } from '@src/new-features/Dashboard';

const App = () => {
  return (
    <div>
      <h1>App</h1>
    </div>
  );
};

const protectedRoutes = [
  {
    path: '/new-app/:appSlug',
    element: <App />,
    children: [
      { path: '/dashboard', element: <Dashboard /> }
    ]
  }
]

export { protectedRoutes };
