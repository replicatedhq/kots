import { useMutation, useQueryClient } from "@tanstack/react-query";

async function postPreflightRun({
  apiEndpoint = process.env.API_ENDPOINT,
  slug,
  sequence,
}: {
  apiEndpoint?: string;
  slug: string;
  sequence: string;
}) {
  const response = await fetch(
    `${apiEndpoint}/app/${slug}/sequence/${sequence}/preflight/run`,
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
      `Encountered an error while fetching preflight results: Unexpected status code: ${response.status}`
    );
  }
}

function useRerunPreflights({
  slug,
  sequence,
}: {
  slug: string;
  sequence: string;
  isUpgradeService?: boolean;
}) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => postPreflightRun({ slug, sequence }),
    onError: (err: Error) => {
      console.log(err);
      throw new Error(err.message || "Error running preflight checks");
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["preflight-results", sequence, slug],
      });
    },
  });
}

export { useRerunPreflights };
