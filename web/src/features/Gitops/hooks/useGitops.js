import { useQuery } from "react-query";
import { Utilities } from "../../../utilities/utilities";

async function getGitops({
  apiEndpoint = process.env.API_ENDPOINT,
  _fetch = fetch,
} = {}) {
  try {
    const res = await _fetch(`${apiEndpoint}/gitops/get`, {
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
      method: "GET",
    });
    if (!res.ok) {
      if (res.status === 401) {
        Utilities.logoutUser();
        return;
      }
      console.log(
        "failed to get gitops settings, unexpected status code",
        res.status
      );
      return;
    }
    return await res.json();
  } catch (err) {
    console.log(err);
    throw err;
  }
}

export default function useGitops() {
  return useQuery(["gitops"], () => getGitops());
}
