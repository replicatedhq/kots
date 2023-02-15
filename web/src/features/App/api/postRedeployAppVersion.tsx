import { useMutation } from "react-query";
import { useHistory } from "react-router-dom";
import { Utilities } from "@src/utilities/utilities";

async function postRedeployAppVersion({
  apiEndpoint = process.env.API_ENDPOINT,
  slug,
  sequence,
}: {
  apiEndpoint?: string;
  slug: string;
  sequence: string;
}) {
  const response = await fetch(
    `${apiEndpoint}/app/${slug}/sequence/${sequence}/redeploy`,
    {
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
        Authorization: Utilities.getToken(),
      },
      method: "POST",
    }
  );

  if (!response.ok) {
    throw new Error(
      `Encountered an error while trying to redeploy downstream version: ${response.status}`
    );
  }
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
    mutationFn: () =>
    postRedeployAppVersion({
        slug,
        sequence,
      }),
    onError: (err: Error) => {
      console.log(err);
      throw new Error(
        err.message ||
          "Encountered an error while trying to redeploy downstream version"
      );
    },
    onSuccess: () => {
      // TODO: refetch useApps (invalidate queries)
    },
  });
}

export { useDeployAppVersion };
