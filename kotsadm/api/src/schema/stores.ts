import { SessionStore } from "../session/session_store";
import { UserStore } from "../user";
import { ClusterStore } from "../cluster";
import { FeatureStore } from "../feature/feature_store";
import { GithubNonceStore } from "../user/store";
import { HealthzStore } from "../healthz/healthz_store";
import { HelmChartStore } from "../helmchart";
import { SnapshotsStore } from "../snapshots";
import { TroubleshootStore } from "../troubleshoot";
import { KotsLicenseStore } from "../klicenses";
import { PreflightStore } from "../preflight/preflight_store";
import { KotsAppStore } from "../kots_app/kots_app_store";
import { KotsAppStatusStore } from "../kots_app/kots_app_status_store";
import { KurlStore } from "../kurl/kurl_store";
import { MetricStore } from "../monitoring/metric_store";
import { ParamsStore } from "../params/params_store";

export interface Stores {
  sessionStore: SessionStore;
  userStore: UserStore;
  githubNonceStore: GithubNonceStore;
  clusterStore: ClusterStore;
  featureStore: FeatureStore;
  healthzStore: HealthzStore;
  helmChartStore: HelmChartStore;
  snapshotsStore: SnapshotsStore,
  troubleshootStore: TroubleshootStore;
  kotsLicenseStore: KotsLicenseStore;
  preflightStore: PreflightStore;
  kotsAppStore: KotsAppStore;
  kotsAppStatusStore: KotsAppStatusStore;
  kurlStore: KurlStore;
  metricStore: MetricStore;
  paramsStore: ParamsStore;
}
