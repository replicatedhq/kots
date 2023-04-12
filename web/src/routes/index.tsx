import React from 'react';
import { useRoutes } from 'react-router-dom-6';

import { protectedRoutes } from './protected';


const AppRoutes = () => {
  // TODO: add auth hook and public routes

  const routes = protectedRoutes;

  const element = useRoutes([...routes]);

  return <>{element}</>;
};

export { AppRoutes }