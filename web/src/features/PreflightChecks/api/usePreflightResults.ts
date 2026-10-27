import { useQuery } from "@tanstack/react-query";
import { PreflightCheck, PreflightResponse, getPreflightDisplayResults, shouldPollPreflightResults } from "../types";
import { useState } from "react";

async function getPreflightResult(url: string): Promise<PreflightResponse> {
  const jsonResponse = await fetch(url, {
    method: "GET",
    credentials: "include",
  });

  if (!jsonResponse.ok) {
    throw new Error(
      `Encountered an error while fetching preflight results: Unexpected status code: ${jsonResponse.status}`
    );
  }

  try {
    const response: PreflightResponse = await jsonResponse.json();

    // unmarshall these nested JSON strings
    if (typeof response?.preflightResult?.result === "string") {
      response.preflightResult.result = JSON.parse(response.preflightResult.result || "{}");
    }

    if (typeof response?.preflightProgress === "string") {
      response.preflightProgress = JSON.parse(response.preflightProgress || "{}");
    }

    return response;
  } catch (err) {
    console.error(err);
    throw new Error(
      "Encountered an error while unmarshalling preflight results"
    );
  }
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

  const result = response?.preflightResult?.result;
  const skipped = response?.preflightResult?.skipped;
  const blocked = response?.preflightResult?.hasFailingStrictPreflights;
  const errors = result?.errors || [];
  const displayResults = getPreflightDisplayResults(response);
  const running = Object.keys(result || {}).length === 0;
  const needsAcknowledgement = !skipped && (
    errors.length > 0 || displayResults.some(result => result?.isFail || result?.isWarn)
  );

  return {
    errors: errors.map(error => error.error),
    pendingPreflightCheckName: response?.preflightProgress?.currentName || "",
    // TODO: see if we can calculate a real %
    pendingPreflightChecksPercentage:
      refetchCount === 0 ? 0 : refetchCount > 21 ? 96 : refetchCount * 4.5,
    pollForUpdates: shouldPollPreflightResults(response),
    preflightResults:
      displayResults.map((responseResult) => ({
        learnMoreUri: responseResult.uri || "",
        message: responseResult.message || "",
        title: responseResult.title || "",
        showCannotFail:
          (responseResult.isFail && responseResult?.strict) || false,
        showFail: responseResult?.isFail || false,
        showPass: responseResult?.isPass || false,
        showWarn: responseResult?.isWarn || false,
      })),
    showCancelPreflight: needsAcknowledgement,
    shouldShowConfirmContinueWithFailedPreflights: needsAcknowledgement,
    shouldShowRerunPreflight: Boolean(result) || skipped,
    showDeploymentBlocked: blocked,
    showIgnorePreflight: (!blocked && skipped) || running,
    showPreflightCheckPending: skipped || running,
    showPreflightResultErrors: errors.length > 0 && !skipped,
    showPreflightResults: !skipped && displayResults.length > 0 && errors.length === 0,
    showPreflightSkipped: skipped,
    showRbacError: errors.some(error => error?.isRbac),
  };
}

export function usePreflightResults({
  url,
  queryKey,
  enabled = true,
  staleTime,
}: {
  url: string;
  queryKey: (string | undefined)[];
  enabled?: boolean;
  staleTime?: number;
}) {
  // this is for the progress bar
  const [refetchCount, setRefetchCount] = useState(0);

  return useQuery({
    queryFn: () => {
      setRefetchCount(refetchCount + 1);

      return getPreflightResult(url);
    },
    queryKey,
    enabled,
    onError: (err: Error) => {
      console.log(err);

      setRefetchCount(0);
    },
    refetchInterval: (preflightCheck: PreflightCheck | undefined) => {
      if (!preflightCheck) return null;

      const refetchInterval = preflightCheck.pollForUpdates ? 1000 : false;

      if (!refetchInterval) setRefetchCount(0);

      return refetchInterval;
    },
    select: (response: PreflightResponse) =>
      flattenPreflightResponse({ response, refetchCount }),
    staleTime,
  });
}
