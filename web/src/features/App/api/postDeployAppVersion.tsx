import { useMutation } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";

async function postDeployAppVersion({
  apiEndpoint = process.env.API_ENDPOINT,
  slug,
  sequence,
  isUpgradeService = false,
  body,
}: {
  apiEndpoint?: string;
  body: string;
  slug: string;
  sequence: string;
  isUpgradeService?: boolean;
}) {
  const url = isUpgradeService
    ? `${process.env.API_ENDPOINT}/upgrade-service/app/${slug}/deploy`
    : `${apiEndpoint}/app/${slug}/sequence/${sequence}/deploy`;
  const response = await fetch(url, {
    headers: {
      "Content-Type": "application/json",
      Accept: "application/json",
    },
    credentials: "include",
    method: "POST",
    body,
  });

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
  isUpgradeService,
}: {
  slug: string;
  sequence: string;
  isUpgradeService?: boolean;
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
        isUpgradeService,
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
