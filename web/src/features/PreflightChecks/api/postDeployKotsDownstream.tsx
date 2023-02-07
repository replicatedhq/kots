import { useMutation } from "react-query";
import { useHistory } from "react-router-dom";
import { Utilities } from "@src/utilities/utilities";

async function postDeployKotsDownstream({
  apiEndpoint = process.env.API_ENDPOINT,
  slug,
  sequence,
}: {
  apiEndpoint?: string;
  slug: string;
  sequence: string;
}) {
  const response = await fetch(
    `${apiEndpoint}/app/${slug}/sequence/${sequence}/deploy`,
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
      `Encountered an error while trying to ignore permissions: ${response.status}`
    );
  }
}

function useDeployKotsDownsteam({
  slug,
  sequence,
}: {
  slug: string;
  sequence: string;
}) {
  const history = useHistory();

  return useMutation({
    mutationFn: () => postDeployKotsDownstream({ slug, sequence }),
    onError: (err: Error) => {
      console.log(err);
      throw new Error(err.message || "Error running preflight checks");
    },
    onSuccess: () => {
      history.push(`/app/${slug}`);
    },
  });
}

export { useDeployKotsDownsteam };

/*

  const deployKotsDownstream = async (
    continueWithFailedPreflights = false,
    isSkipPreflights = false
  ) => {
    setState({ errorMessage: "" });
    try {
      const { match } = props;
      const { slug } = match.params;
      const { preflightResultData } = state;

      if (!isSkipPreflights) {
        const preflightResults = JSON.parse(preflightResultData?.result || "");
        const preflightState = getPreflightResultState(preflightResults);
        if (preflightState !== "pass") {
          if (!continueWithFailedPreflights) {
            this.showWarningModal();
            return;
          }
        }
      }

      const sequence = match.params.sequence
        ? parseInt(match.params.sequence, 10)
        : 0;
      await fetch(
        `${process.env.API_ENDPOINT}/app/${slug}/sequence/${sequence}/deploy`,
        {
          headers: {
            Authorization: Utilities.getToken(),
            "Content-Type": "application/json",
          },
          method: "POST",
          body: JSON.stringify({
            isSkipPreflights: isSkipPreflights,
            continueWithFailedPreflights: !!continueWithFailedPreflights,
          }),
        }
      );

      history.push(`/app/${slug}`);
    } catch (err) {
      console.log(err);
      const errorMessage =
        err instanceof Error ? err.message : "Something went wrong";
      setState({
        errorMessage: err
          ? `Encountered an error while trying to deploy downstream version: ${errorMessage}`
          : "Something went wrong, please try again.",
      });
    }
  };
  */
