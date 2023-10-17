import { useQuery } from '@tanstack/react-query';

async function getMetadata({
  apiEndpoint = process.env.API_ENDPOINT,
  _fetch = fetch,
} = {}) {
  try {
    const res = await _fetch(`${apiEndpoint}/metadata`, {
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
      method: "GET",
    });
    return await res.json();
  } catch (err) {
    throw Error(err);
  }
}

function useMetadata({ _getMetadata = getMetadata } = {}) {
  return useQuery(['metadata'], () => _getMetadata(), {
    staleTime: Infinity,
  });
}

export { getMetadata, useMetadata };
