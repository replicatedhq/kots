import { useQuery } from "react-query";
import { Utilities } from "../../../utilities/utilities";

async function getApps({
  accessToken = Utilities.getToken(),
  apiEndpoint = process.env.API_ENDPOINT,
  _fetch = fetch,
} = {}) {
  try {
    const res = await _fetch(`${apiEndpoint}/apps`, {
      headers: {
        Authorization: accessToken,
        "Content-Type": "application/json",
      },
      method: "GET",
    });
    if (!res.ok) {
      if (res.status === 401) {
        Utilities.logoutUser();
        return null;
      }
      throw Error(`Failed to fetch apps with status ${res.status}`);
    }
    return await res.json();
  } catch (err) {
    throw Error(err);
  }
}

function useApps({ _getApps = getApps } = {}) {
  return useQuery("apps", () => _getApps(), {
    staleTime: 5000,
  });
}

function UseApps({ children }) {
  const query = useApps();

  return children(query);
}

export { getApps, useApps, UseApps };
