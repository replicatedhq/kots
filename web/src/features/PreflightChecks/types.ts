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
