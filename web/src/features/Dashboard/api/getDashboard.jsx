// This hook has not been integrated yet.
import { useQuery } from "react-query";
import { Utilities } from "../../../utilities/utilities";

async function getDashbaord({
  appSlug,
  clusterId,
  accessToken = Utilities.getToken(),
  apiEndpoint = process.env.API_ENDPOINT,
  _fetch = fetch,
} = {}) {
  try {
    let response = await _fetch(
      `${process.env.API_ENDPOINT}/app/${appSlug}/cluster/${clusterId}/dashboard`,
      {
        headers: {
          Authorization: Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "GET",
      }
    );
    if (!response.ok && response.status === 401) {
      Utilities.logoutUser();
      return;
    }
    return await response.json();
  } catch (err) {
    throw Error(err);
  }
}

function useDashboard({
  appSlug,
  clusterId,
  refetchInterval,
  _getDashboard = getDashboard,
} = {}) {
  return useQuery(
    ["apps", appSlug, clusterId],
    () => _getDashboard({ appSlug, clusterId }),
    {
      refetchInterval,
    }
  );
}

export { getDashbaord, useDashboard };
