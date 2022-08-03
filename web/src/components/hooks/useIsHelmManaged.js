import { useQuery } from "react-query";
import { Utilities } from "../../utilities/utilities";

async function fetchIsHelmManaged({
  accessToken = Utilities.getToken(),
  apiEndpoint = process.env.API_ENDPOINT,
  _fetch = fetch,
} = {}) {
  try {
    const res = await _fetch(`${apiEndpoint}/is-helm-managed`, {
      headers: {
        Authorization: accessToken,
        "Content-Type": "application/json",
      },
      method: "GET",
    });
    if (res.ok) {
      const response = await res.json();
      return { isHelmManaged: response.isHelmManaged };
    }
    throw new Error("Error fetching isHelmManaged");
  } catch (err) {
    throw Error(err);
  }
}

function useIsHelmManaged({ _fetchIsHelmManaged = fetchIsHelmManaged } = {}) {

  // const { isHelmManaged } = data;
  return useQuery("isHelmManaged", () => _fetchIsHelmManaged(), {
    staleTime: Infinity,
  });
}

function UseIsHelmManaged({ children }) {
  const query = useIsHelmManaged();

  return children(query);
}

export { UseIsHelmManaged, fetchIsHelmManaged, useIsHelmManaged };
