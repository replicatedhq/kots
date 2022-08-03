import { useQuery } from "react-query";
import { Utilities } from "@src/utilities/utilities";

async function getMetadata({
  accessToken = Utilities.getToken(),
  apiEndpoint = process.env.API_ENDPOINT,
  _fetch = fetch,
} = {}) {
  try {
    const res = await _fetch(`${apiEndpoint}/metadata`, {
      headers: {
        Authorization: accessToken,
        "Content-Type": "application/json",
      },
      method: "GET",
    });
    return await res.json();
  } catch (err) {
    throw Error(err);
  }
}

function useMetadata({ _getMetadata = getMetadata } = {}) {
  return useQuery("metadata", () => _getMetadata(), {
    staleTime: Infinity,
  });
}

export { getMetadata, useMetadata };