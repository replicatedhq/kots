import { useMutation, useQueryClient } from "@tanstack/react-query";

async function postPreflightRun({ slug }: { slug: string }) {
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

function useRunPreflights({
  slug,
  sequence,
}: {
  slug: string;
  sequence: string;
}) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => postPreflightRun({ slug }),
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

export { useRunPreflights };
