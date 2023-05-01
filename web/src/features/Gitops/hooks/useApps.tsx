import { useQuery } from "react-query";
import { Utilities } from "../../../utilities/utilities";

// TODO: replace with fetatures/App/api
async function getApps({
  apiEndpoint = process.env.API_ENDPOINT,
  _fetch = fetch,
} = {}) {
  try {
    const res = await _fetch(`${apiEndpoint}/apps`, {
      headers: {
        "Content-Type": "application/json",
      },
      method: "GET",
      credentials: "include",
    });
    if (!res.ok) {
      if (res.status === 401) {
        Utilities.logoutUser();
        return null;
      }
      throw Error(`Failed to fetch apps with status ${res.status}`);
    }
    const { apps } = await res.json();

    return apps;
  } catch (err) {
    if (err instanceof Error) throw Error(err.message);

    throw Error("Failed to fetch apps");
  }
}

export default function useApps() {
  return useQuery(["apps"], () => getApps());
}
