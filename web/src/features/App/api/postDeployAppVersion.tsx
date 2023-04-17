import { useMutation } from "react-query";
import { useHistory } from "react-router-dom";

async function postDeployAppVersion({
  apiEndpoint = process.env.API_ENDPOINT,
  slug,
  sequence,
  body,
}: {
  apiEndpoint?: string;
  body: string;
  slug: string;
  sequence: string;
}) {
  const response = await fetch(
    `${apiEndpoint}/app/${slug}/sequence/${sequence}/deploy`,
    {
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      credentials: "include",
      method: "POST",
      body,
    }
  );

  if (!response.ok) {
    throw new Error(
      `Encountered an error while trying to deploy downstream version: ${response.status}`
    );
  }
}

function makeBody({
  continueWithFailedPreflights,
  isSkipPreflights,
}: {
  continueWithFailedPreflights: boolean;
  isSkipPreflights: boolean;
}) {
  return JSON.stringify({
    continueWithFailedPreflights,
    isSkipPreflights,
  });
}

function useDeployAppVersion({
  slug,
  sequence,
}: {
  slug: string;
  sequence: string;
}) {
  const history = useHistory();

  return useMutation({
    mutationFn: ({
      continueWithFailedPreflights = false,
      isSkipPreflights = false,
    }: {
      continueWithFailedPreflights?: boolean;
      isSkipPreflights?: boolean;
    }) =>
      postDeployAppVersion({
        slug,
        sequence,
        body: makeBody({ continueWithFailedPreflights, isSkipPreflights }),
      }),
    onError: (err: Error) => {
      console.log(err);
      throw new Error(
        err.message ||
          "Encountered an error while trying to deploy downstream version"
      );
    },
    onSuccess: () => {
      history.push(`/app/${slug}`);
    },
  });
}

export { useDeployAppVersion };
