import { useMutation } from "react-query";

async function postUpdateAdminConsole({
  apiEndpoint = process.env.API_ENDPOINT,
  slug,
  sequence,
}: {
  apiEndpoint?: string;
  slug: string;
  sequence: string;
}) {
  const response = await fetch(
    `${apiEndpoint}/app/${slug}/sequence/${sequence}/update-console`,
    {
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      credentials: "include",
      method: "POST",
    }
  );

  if (!response.ok) {
    throw new Error(
      `Error while trying to update admin console: ${response.status}`
    );
  }

  try {
    return await response.json();
  } catch (err) {
    if (err instanceof Error) {
      throw new Error(
        `Error while trying to unmarshal update admin console response: ${err.message}`
      );
    }
    throw new Error(
      `Error while trying to unmarshal update admin console response`
    );
  }
}

function useUpdateAdminConsole({
  slug,
  sequence,
}: {
  slug: string;
  sequence: string;
}) {
  // TODO: add refetching behavior that uses getAdminConsoleUpdateStatus
  return useMutation({
    mutationFn: () =>
      postUpdateAdminConsole({
        slug,
        sequence,
      }),
    onError: (err: Error) => {
      console.log(err);
      throw new Error(
        err.message ||
          "Error while trying to update admin console. Please try again."
      );
    },
    onSuccess: () => {
      // TODO: delete getKotsUpdateStatus query
    },
  });
}

export { useUpdateAdminConsole };
