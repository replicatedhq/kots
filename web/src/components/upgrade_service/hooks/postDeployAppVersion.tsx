import { useMutation } from "@tanstack/react-query";

async function postDeployAppVersion({
  slug,
  body,
}: {
  body: string;
  slug: string;
}) {
  const url = `${process.env.API_ENDPOINT}/upgrade-service/app/${slug}/deploy`;
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
      `Encountered an error while trying to deploy app version: ${response.status}`
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
  closeModal,
}: {
  slug: string;
  closeModal: () => void;
}) {
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
      closeModal();
    },
  });
}

export { useDeployAppVersion };
