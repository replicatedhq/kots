import { useQuery } from "react-query";
import { Utilities } from "@src/utilities/utilities";
import { PreflightCheck, PreflightResponse } from "../types";
import { useState } from "react";

async function getPreflightResult({
  apiEndpoint = process.env.API_ENDPOINT,
  slug,
  sequence,
}: {
  apiEndpoint?: string;
  slug: string;
  sequence?: number;
}): Promise<PreflightResponse> {
  const getUrl = sequence
    ? `${apiEndpoint}/app/${slug}/sequence/${sequence}/preflight/result`
    : `${apiEndpoint}/app/${slug}/preflight/result`;
  const jsonResponse = await fetch(getUrl, {
    method: "GET",
    headers: {
      Authorization: Utilities.getToken(),
    },
  });

  if (!jsonResponse.ok) {
    throw new Error(
      `Encountered an error while fetching preflight results: Unexpected status code: ${jsonResponse.status}`
    );
  }

  const response: PreflightResponse = await jsonResponse.json();

  // unmarshall these nested JSON strings
  if (typeof response?.preflightResult?.result === "string")
    response.preflightResult.result = JSON.parse(
      response.preflightResult.result
    );

  if (typeof response?.preflightProgress === "string")
    response.preflightProgress = JSON.parse(response.preflightProgress);

  return response;
}

function flattenPreflightResponse({
  refetchCount,
  response,
}: {
  refetchCount: number;
  response: PreflightResponse;
}): PreflightCheck {
  if (
    typeof response?.preflightProgress === "string" ||
    typeof response?.preflightResult?.result === "string"
  )
    throw new Error("Preflight response is not properly unmarshalled");

  return {
    // flatten the error strings out into an array
    errors:
      response?.preflightResult?.result?.errors?.map((error) => error.error) ||
      [],
    pendingPreflightCheckName: response?.preflightProgress?.currentName || "",
    // TODO: see if we can calculate a real %
    pendingPreflightChecksPercentage:
      refetchCount === 0 ? 0 : refetchCount > 21 ? 96 : refetchCount * 4.5,
    preflightResults:
      response?.preflightResult?.result?.results?.map((responseResult) => ({
        learnMoreUri: responseResult.uri || "",
        message: responseResult.message || "",
        title: responseResult.title || "",
        showCannotFail: responseResult?.strict || false,
        showFail: responseResult?.isFail || false,
        showPass: responseResult?.isPass || false,
        showWarn: responseResult?.isWarn || false,
      })) || [],
    showDeploymentBlocked: response?.preflightResult?.result?.results?.find(
      (result) => result?.isFail && result?.strict
    )
      ? true
      : false,
    showPreflightCheckPending: !response?.preflightResult?.result,
    showPreflightNoChecks:
      response?.preflightResult?.result?.results?.length === 0,
    showPreflightSkipped: response?.preflightResult?.skipped,
    showRbacError: response?.preflightResult?.result?.errors?.find(
      (error) => error?.isRbac
    )
      ? true
      : false,
  };
}

function makeRefetchInterval(preflightCheck: PreflightCheck): number | false {
  if (preflightCheck.showPreflightCheckPending) return 1000;

  return false;
}

function useGetPrelightResults({
  slug,
  sequence,
}: {
  slug: string;
  sequence?: number;
}) {
  // this is for the progress bar
  const [refetchCount, setRefetchCount] = useState(0);

  return useQuery({
    queryFn: () => {
      setRefetchCount(refetchCount + 1);

      return getPreflightResult({ slug, sequence });
    },
    queryKey: ["preflight-results", slug, sequence],
    onError: (err: Error) => {
      console.log(err);

      setRefetchCount(0);
    },
    refetchInterval: (preflightCheck: PreflightCheck | undefined) => {
      if (!preflightCheck) return false;

      const refetchInterval = makeRefetchInterval(preflightCheck);

      if (!refetchInterval) setRefetchCount(0);

      return refetchInterval;
    },
    select: (response: PreflightResponse) =>
      flattenPreflightResponse({ response, refetchCount }),
    staleTime: 500,
  });
}

export { useGetPrelightResults };
