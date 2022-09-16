import { useCallback, useEffect, useState } from 'react';
import { Utilities } from "@src/utilities/utilities";

function useAuth() {
  // pull current state out of storage
  const [isAuthenticated, setIsAuthenticated] = useState<boolean | null>(null);

  useEffect(() => {
    if (isAuthenticated === null) {
      const token = Utilities.getToken();

      if (token) {
        setIsAuthenticated(true);
      } else {
        setIsAuthenticated(false);
      }
    }

    if (isAuthenticated === false) {
      Utilities.logoutUser();
    }

  }, [isAuthenticated]);

  const logout = useCallback(() => {
    setIsAuthenticated(false);
  }, [isAuthenticated])

  return {
    isAuthenticated,
    logout
  };
}

export { useAuth };