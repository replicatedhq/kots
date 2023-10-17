import { useQuery } from '@tanstack/react-query';

async function getAdminConsoleUpdateStatus({
  apiEndpoint = process.env.API_ENDPOINT,
  sequence,
  slug,
}: {
  apiEndpoint?: string;
  sequence: string;
  slug: string;
}) {
  const response = await fetch(
    `${apiEndpoint}/app/${slug}/sequence/${sequence}/download`,
    {
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      credentials: "include",
      method: "GET",
    }
  );

  if (!response.ok) {
    throw new Error(
      `Error while trying to download version sequence ${sequence}: ${response.status}`
    );
  }

  try {
    return await response.json();
  } catch (err) {
    if (err instanceof Error) {
      throw new Error(
        `Error while trying to unmarshal download version status: ${err.message}`
      );
    }
    throw new Error(`Error while trying to unmarshal download version status`);
  }
}

function useAdminConsoleUpdateStatus({
  sequence,
  slug,
}: {
  sequence: string;
  slug: string;
}) {
  return useQuery({
    queryFn: () =>
      getAdminConsoleUpdateStatus({
        sequence,
        slug,
      }),
    // TODO: add refetch interval
    onError: (err: Error) => {
      console.log(err);
      throw new Error(
        err.message ||
          "Error while trying to get admin console update status. Please try again."
      );
    },
  });
}

export { useAdminConsoleUpdateStatus };
