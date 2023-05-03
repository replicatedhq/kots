import { useMutation } from "react-query";

async function postDownloadAppVersion({
  apiEndpoint = process.env.API_ENDPOINT,
  slug,
  sequence,
}: {
  apiEndpoint?: string;
  slug: string;
  sequence: string;
}) {
  const response = await fetch(
    `${apiEndpoint}/app/${slug}/sequence/${sequence}/download`,
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
      `Encountered an error while trying download version: ${response.status}`
    );
  }
}

function useDownloadAppVersion({
  slug,
  sequence,
}: {
  slug: string;
  sequence: string;
}) {
  return useMutation({
    mutationFn: () =>
      postDownloadAppVersion({
        slug,
        sequence,
      }),
    onError: (err: Error) => {
      console.log(err);
      throw new Error(
        err.message || "Encountered an error while trying to download version"
      );
    },
  });
}

export { useDownloadAppVersion };
