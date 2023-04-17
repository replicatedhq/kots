import { useQuery } from "react-query";
import { Utilities } from "@src/utilities/utilities";

async function getAdminConsoleUpdateStatus({
  apiEndpoint = process.env.API_ENDPOINT,
  slug,
}: {
  apiEndpoint?: string;
  slug: string;
}) {
  const response = await fetch(
    `${apiEndpoint}/app/${slug}/update-admin-console`,
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
      `Error while trying to get admin console update status: ${response.status}`
    );
  }

  try {
    return await response.json();
  } catch (err) {
    if (err instanceof Error) {
      throw new Error(
        `Error while trying to unmarshal get admin update status: ${err.message}`
      );
    }
    throw new Error(`Error while trying to unmarshal get admin update status`);
  }
}

function useAdminConsoleUpdateStatus({ slug }: { slug: string }) {
  return useQuery({
    queryFn: () =>
      getAdminConsoleUpdateStatus({
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
