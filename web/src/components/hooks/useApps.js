import { useQuery } from "react-query";
import { Utilities } from "../../utilities/utilities";

async function fetchApps({
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
        return;
      }
      throw Error(`Failed to fetch apps with status ${res.status}`);
    }
    const response = await res.json();
    return response;
  } catch (err) {
    throw Error(err);
  }
}

function useApps({ _fetchApps = fetchApps} = {}) {
  return useQuery("apps", () => _fetchApps(), {
    staleTime: 5000,
  });
}

function UseApps({ children }) {
  const query = useApps();

  return children(query);
}

export { fetchApps, useApps, UseApps };