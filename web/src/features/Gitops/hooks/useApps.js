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
    const { apps } = await res.json();

    return apps;
  } catch (err) {
    throw Error(err);
  }
}

export default function useApps() {
  return useQuery(["apps"], () => getApps());
}
