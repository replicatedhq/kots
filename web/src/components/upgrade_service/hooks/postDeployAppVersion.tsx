import { useMutation } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";

async function postDeployAppVersion({
  slug,
  body,
}: {
  apiEndpoint?: string;
  body: string;
  slug: string;
  sequence: string;
}) {
  const response = await fetch(
    `${process.env.API_ENDPOINT}/upgrade-service/app/${slug}/deploy`,
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
  const navigate = useNavigate();

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
      // TODO: figure out how to close the modal
      navigate(`/app/${slug}`);
    },
  });
}

export { useDeployAppVersion };
