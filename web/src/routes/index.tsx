import React from 'react';
import { useRoutes } from 'react-router-dom';

import { protectedRoutes } from './protected';


const AppRoutes = () => {
  // TODO: add auth hook and public routes

  const commonRoutes = [{ path: '/', element: <div>Home</div> }];

  const routes = protectedRoutes;

  const element = useRoutes([...routes, ...commonRoutes]);

  return <>{element}</>;
};

export { AppRoutes }