import { useMutation } from "react-query";
import { Utilities } from "@src/utilities/utilities";

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
        Authorization: Utilities.getToken(),
        "Content-Type": "application/json",
      },
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