import { useMutation, useQueryClient } from "@tanstack/react-query";

async function postPreflightRun({
  slug,
}: {
  apiEndpoint?: string;
  slug: string;
  sequence: string;
}) {
  const response = await fetch(
    `${process.env.API_ENDPOINT}/upgrade-service/app/${slug}/preflight/run`,
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
