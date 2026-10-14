/* REQUEST TYPES */

/* RESPONSE TYPES */
export interface PreflightResponse {
  preflightProgress?: PreflightResponseProgress | string; // only returned when preflight is running
  preflightResult: PreflightResponseResult;
}

export type PreflightResponseProgress = {
  completedCount?: number;
  currentName?: string;
  currentStatus?: string;
  totalCount?: number;
  updatedAt?: string;
};

interface PreflightResponseResult {
  appSlug: string;
  clusterSlug: string;
  createdAt: string; // ISO 8601
  hasFailingStrictPreflights: boolean;
  skipped: boolean;
  result:
    | {
        notice?: string;
        errors?: PreflightResponseError[];
        results?: PreflightResponseResultItem[];
      }
    | string;
}

interface PreflightResponseError {
  error: string;
  isRbac: boolean;
}

interface PreflightResponseResultItem {
  // has one isFail isPass isWarn
  isFail?: true;
  isPass?: true;
  isWarn?: true;
  message: string;
  strict: boolean;
  title: string;
  uri: string;
}

// Add the notice only for display; it is never a saved analyzer result.
export function getPreflightDisplayResults(
  response: PreflightResponse
): PreflightResponseResultItem[] {
  const result = response?.preflightResult?.result;
  if (typeof result === "string")
    throw new Error("Preflight response is not properly unmarshalled");
  const results = result?.results || [];
  return result?.notice
    ? [
        { isWarn: true, title: "Unsupported v1beta3 specs", message: result.notice, strict: false, uri: "" },
        ...results,
      ]
    : results;
}

/* UI TYPES */

export interface PreflightResult {
  learnMoreUri: string;
  message: string;
  title: string;
  showCannotFail: boolean;
  showFail: boolean;
  showPass: boolean;
  showWarn: boolean;
}

export interface PreflightCheck {
  errors: string[];
  pendingPreflightCheckName: string;
  pendingPreflightChecksPercentage: number;
  pollForUpdates: boolean;
  preflightResults: PreflightResult[];
  shouldShowConfirmContinueWithFailedPreflights: boolean;
  shouldShowRerunPreflight: boolean;
  showCancelPreflight: boolean;
  showDeploymentBlocked: boolean;
  showIgnorePreflight: boolean;
  showPreflightCheckPending: boolean;
  showPreflightResultErrors: boolean;
  showPreflightResults: boolean;
  showPreflightSkipped: boolean;
  showRbacError: boolean;
}
