import React from 'react';
import { Navigate, Outlet } from 'react-router-dom';

import { Dashboard } from '@src/new-features/Dashboard';

const App = () => {
  return (
    <div>
      <h1>App</h1>
      <Outlet />
    </div>
  );
};

const protectedRoutes = [
  { path: '/', element: <Navigate to="/app/dashboard" /> },
  {
    //    path: '/app/:appSlug',
    path: '/app',
    element: <App />,
    children: [
      { path: 'dashboard', element: <Dashboard /> }
    ]
  }
]

export { protectedRoutes };
