import { useMutation, useQueryClient } from "react-query";

async function postIgnorePermissionErrors({
  apiEndpoint = process.env.API_ENDPOINT,
  slug,
  sequence,
}: {
  apiEndpoint?: string;
  slug: string;
  sequence: string;
}) {
  const response = await fetch(
    `${apiEndpoint}/app/${slug}/sequence/${sequence}/preflight/ignore-rbac`,
    {
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      credentials: "include",
      method: "POST",
    }
  );

  if (!response.ok) {
    throw new Error(
      `Encountered an error while trying to ignore permissions: ${response.status}`
    );
  }

  return response;
}

function useIgnorePermissionErrors({
  slug,
  sequence,
}: {
  slug: string;
  sequence: string;
}) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => postIgnorePermissionErrors({ slug, sequence }),
    onError: (err: Error) => {
      console.log(err);
      throw new Error(
        err.message || "Encountered an error while trying to ignore permissions"
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["preflight-results", sequence, slug],
      });
    },
  });
}

export { useIgnorePermissionErrors };
