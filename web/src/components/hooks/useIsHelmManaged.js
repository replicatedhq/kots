import { useState, useEffect } from "react";
import { singletonHook } from "react-singleton-hook";
import { Utilities } from "../../utilities/utilities";

async function fetchIsHelmManaged({
  accessToken = Utilities.getToken(),
  apiEndpoint = process.env.API_ENDPOINT,
  signal = null,
} = {}) {
  try {
    const res = await fetch(`${apiEndpoint}/isHelmManaged`, {
      headers: {
        Authorization: accessToken,
        "Content-Type": "application/json",
      },
      method: "GET",
      signal,
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

function useIsHelmManagedImpl({
  _fetchIsHelmManaged = fetchIsHelmManaged,
} = {}) {
  const [isHelmManaged, setIsHelmManaged] = useState(null);
  const [isHelmManagedLoading, setIsHelmManagedLoading] = useState(false);

  useEffect(() => {
    let controller;
    if (isHelmManaged === null) {
      controller = new AbortController();
      setIsHelmManaged(false);
      setIsHelmManagedLoading(true);
      _fetchIsHelmManaged({ signal: controller.signal }).then(
        ({ isHelmManaged: _isHelmManaged }) => {
          setIsHelmManaged(_isHelmManaged);
          setIsHelmManagedLoading(false);
        }
      );
    }

    return () => {
      if (controller) {
        controller.abort();
      }
    };
  }, []);

  return {
    isHelmManaged,
    isHelmManagedLoading,
  };
}

const useIsHelmManaged = singletonHook(
  { isHelmManaged: null, isHelmManagedLoading: false },
  useIsHelmManagedImpl
);

function IsHelmManaged({ children }) {
  const { isHelmManaged, isHelmManagedLoading } = useIsHelmManaged();

  return children({
    isHelmManaged,
    isHelmManagedLoading,
  });
}

export { IsHelmManaged, fetchIsHelmManaged, useIsHelmManaged };
