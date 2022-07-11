import { useState, useEffect } from "react";
import { Utilities } from "../../utilities/utilities";

async function fetchIsHelmManaged({
  accessToken = Utilities.getToken(),
  apiEndpoint = process.env.API_ENDPOINT,
} = {}) {
  try {
    const res = await fetch(`${apiEndpoint}/isHelmManaged`, {
      headers: {
        Authorization: accessToken,
        "Content-Type": "application/json",
      },
      method: "GET",
    });
    if (res.ok && res.status === 200) {
      const response = await res.json();
      return { isHelmManaged: response.isHelmManaged };
    }
    return { isHelmManaged: false };
  } catch (err) {
    console.log(err);
    return { isHelmManaged: false };
  }
}

function useIsHelmManaged({ _fetchIsHelmManaged = fetchIsHelmManaged } = {}) {
  const [isHelmManaged, setIsHelmManaged] = useState(null);
  const [isHelmManagedLoading, setIsHelmManagedLoading] = useState(false);

  useEffect(() => {
    if (isHelmManaged === null) {
      setIsHelmManaged(false);
      setIsHelmManagedLoading(true);
      _fetchIsHelmManaged().then(({ isHelmManaged: _isHelmManaged }) => {
        setIsHelmManaged(_isHelmManaged);
        setIsHelmManagedLoading(false);
      });
    }
  }, []);

  return {
    isHelmManaged,
    isHelmManagedLoading,
  };
}

function IsHelmManaged({ children }) {
  const { isHelmManaged, isHelmManagedLoading } = useIsHelmManaged();

  return children({
    isHelmManaged,
    isHelmManagedLoading,
  });
}

export { IsHelmManaged, fetchIsHelmManaged, useIsHelmManaged };
