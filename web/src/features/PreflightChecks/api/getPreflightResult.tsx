import { useQuery } from "@tanstack/react-query";
import { PreflightCheck, PreflightResponse } from "../types";
import { useState } from "react";

async function getPreflightResult({
  apiEndpoint = process.env.API_ENDPOINT,
  slug,
  sequence,
}: {
  apiEndpoint?: string;
  slug: string;
  sequence?: string;
}): Promise<PreflightResponse> {
  const getUrl = sequence
    ? `${apiEndpoint}/app/${slug}/sequence/${sequence}/preflight/result`
    : `${apiEndpoint}/app/${slug}/preflight/result`;
  const jsonResponse = await fetch(getUrl, {
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
      if (response?.preflightResult?.result.length > 0) {
        response.preflightResult.result = JSON.parse(
          response.preflightResult.result
        );
      } else {
        response.preflightResult.result = {};
      }
    }

    if (typeof response?.preflightProgress === "string") {
      if (response?.preflightProgress.length > 0) {
        response.preflightProgress = JSON.parse(response.preflightProgress);
      } else {
        response.preflightProgress = {};
      }
    }

    return response;
  } catch (err) {
    console.error(err);
    throw new Error(
      "Encountered an error while unmarshalling preflight results"
    );
  }
}

function hasPreflightErrors(response: PreflightResponse): boolean {
  if (typeof response?.preflightResult?.result === "string")
    throw new Error("Preflight response is not properly unmarshalled");

  return Boolean(response?.preflightResult?.result?.errors?.length);
}

// result.results which is an array
function hasPreflightResults(response: PreflightResponse): boolean {
  if (typeof response?.preflightResult?.result === "string")
    throw new Error("Preflight response is not properly unmarshalled");

  return Boolean(response?.preflightResult?.result?.results?.length);
}

// just results which is an object
function hasRunningPreflightChecks(response: PreflightResponse): boolean {
  if (typeof response?.preflightResult?.result === "string")
    throw new Error("Preflight response is not properly unmarshalled");

  return Object.keys(response?.preflightResult?.result || {}).length === 0;
}

function hasFailureOrWarning(response: PreflightResponse): boolean {
  if (typeof response?.preflightResult?.result === "string")
    throw new Error("Preflight response is not properly unmarshalled");

  return Boolean(
    response?.preflightResult?.result?.results?.find(
      (result) => result?.isFail || result?.isWarn
    )
  );
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
    pollForUpdates:
      !response?.preflightResult?.skipped ||
      hasRunningPreflightChecks(response),
    preflightResults:
      response?.preflightResult?.result?.results?.map((responseResult) => ({
        learnMoreUri: responseResult.uri || "",
        message: responseResult.message || "",
        title: responseResult.title || "",
        showCannotFail:
          (responseResult.isFail && responseResult?.strict) || false,
        showFail: responseResult?.isFail || false,
        showPass: responseResult?.isPass || false,
        showWarn: responseResult?.isWarn || false,
      })) || [],
    showCancelPreflight:
      !response?.preflightResult?.skipped &&
      (hasPreflightErrors(response) || hasFailureOrWarning(response)),
    shouldShowConfirmContinueWithFailedPreflights:
      !response?.preflightResult?.skipped && // not skipped
      (hasFailureOrWarning(response) || hasPreflightErrors(response)), // or it has errors
    shouldShowRerunPreflight:
      Boolean(response?.preflightResult?.result) || // not running
      response?.preflightResult?.skipped, // not skipped
    showDeploymentBlocked:
      response?.preflightResult?.hasFailingStrictPreflights,
    showIgnorePreflight:
      (!response?.preflightResult?.hasFailingStrictPreflights &&
        response?.preflightResult?.skipped) ||
      hasRunningPreflightChecks(response),
    showPreflightCheckPending:
      response?.preflightResult?.skipped || hasRunningPreflightChecks(response),
    showPreflightResultErrors:
      hasPreflightErrors(response) && // has errors
      !response?.preflightResult?.skipped && // not skipped
      !hasPreflightResults(response),
    showPreflightResults:
      !response?.preflightResult?.skipped &&
      hasPreflightResults(response) &&
      !hasPreflightErrors(response),
    showPreflightSkipped: response?.preflightResult?.skipped,
    showRbacError: response?.preflightResult?.result?.errors?.find(
      (error) => error?.isRbac
    )
      ? true
      : false,
  };
}

function makeRefetchInterval(preflightCheck: PreflightCheck): number | false {
  if (preflightCheck.pollForUpdates) return 1000;

  return false;
}

function useGetPrelightResults({
  slug,
  sequence,
}: {
  slug: string;
  sequence?: string;
}) {
  // this is for the progress bar
  const [refetchCount, setRefetchCount] = useState(0);

  return useQuery({
    queryFn: () => {
      setRefetchCount(refetchCount + 1);

      return getPreflightResult({ slug, sequence });
    },
    queryKey: ["preflight-results", sequence, slug],
    onError: (err: Error) => {
      console.log(err);

      setRefetchCount(0);
    },
    refetchInterval: (preflightCheck: PreflightCheck | undefined) => {
      if (!preflightCheck) return null;
      if (preflightCheck?.preflightResults.length > 0) return null;

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
