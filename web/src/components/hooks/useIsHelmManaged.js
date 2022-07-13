import { useState, useEffect } from "react";
import { Utilities } from "../../utilities/utilities";
import { useQuery } from "react-query";

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
  return useQuery("isHelmManaged", () => _fetchIsHelmManaged(), {
    staleTime: Infinity,
  });
}

function UseIsHelmManaged({ children }) {
  const query  = useIsHelmManaged();

  return children(query);
}

export { UseIsHelmManaged, fetchIsHelmManaged, useIsHelmManaged };
