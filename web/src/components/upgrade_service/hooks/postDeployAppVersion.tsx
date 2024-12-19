import { useMutation } from "@tanstack/react-query";

async function postDeployAppVersion({
  slug,
  body,
  isEC2Install,
}: {
  body: string;
  slug: string;
  isEC2Install: boolean;
}) {
  let url = `${process.env.API_ENDPOINT}/upgrade-service/app/${slug}/deploy`;
  if (isEC2Install) {
    url = `${process.env.API_ENDPOINT}/upgrade-service/app/${slug}/ec2-deploy`;
  }
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
  isEC2Install,
}: {
  slug: string;
  closeModal: () => void;
  isEC2Install: boolean;
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
        isEC2Install,
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
