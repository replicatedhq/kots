import { UserQueries, UserMutations } from "../user";
import { ClusterMutations, ClusterQueries } from "../cluster";
import { WatchMutations, WatchQueries } from "../watch";
import { UpdateMutations, UpdateQueries } from "../update";
import { UnforkQueries, UnforkMutations } from "../unfork";
import { NotificationQueries, NotificationMutations } from "../notification";
import { InitMutations, InitQueries } from "../init";
import { FeatureMutations, FeatureQueries } from "../feature";
import { EditMutations, EditQueries } from "../edit";
import { PendingQueries } from "../pending";
import { HealthzQueries } from "../healthz";
import { Params } from "../server/params";
import { Stores } from "./stores";
import { GithubInstallationQueries, GithubInstallationMutations } from "../github_installation";
import { HelmChartQueries, HelmChartMutations } from "../helmchart";
import { TroubleshootQueries, TroubleshootMutations } from "../troubleshoot";
import { LicenseQueries, LicenseMutations } from "../license";
import { KotsLicenseQueries, KotsLicenseMutations } from "../klicenses";
import { PrefightQueries } from "../preflight";
import { AppsQueries } from "../apps";
import { KotsQueries, KotsDashboardQueries, KotsMutations } from "../kots_app";
import { KurlQueries, KurlMutations } from "../kurl";

export const Resolvers = (stores: Stores, params: Params) => {
  let query = {
    ...FeatureQueries(stores),
    ...HealthzQueries(stores),
    ...TroubleshootQueries(stores),
    ...PrefightQueries(stores),
    ...AppsQueries(stores),
    ...ClusterQueries(stores),
    ...UserQueries(stores),
  };

  if (params.enableKots) {
    query = {
      ...query,
      ...KotsQueries(stores),
      ...KotsDashboardQueries(stores),
      ...KotsLicenseQueries(stores),
      ...KurlQueries(stores, params),
    }
  }

  if (params.enableShip) {
    query = {
      ...query,
      ...WatchQueries(stores),
      ...UpdateQueries(stores),
      ...UnforkQueries(stores),
      ...NotificationQueries(stores),
      ...InitQueries(stores),
      ...GithubInstallationQueries(stores),
      ...EditQueries(stores),
      ...PendingQueries(stores),
      ...HelmChartQueries(stores),
      ...LicenseQueries(stores),
    }
  }

  let mutation = {
    ...ClusterMutations(stores, params),
    ...FeatureMutations(stores),
    ...TroubleshootMutations(stores, params),
    ...UserMutations(stores, params),
  };

  if (params.enableKots) {
    mutation = {
      ...mutation,
      ...KotsLicenseMutations(stores),
      ...KotsMutations(stores),
      ...KurlMutations(stores, params),
    };
  }

  if (params.enableShip) {
    mutation = {
      ...mutation,
      ...GithubInstallationMutations(stores),
      ...EditMutations(stores),
      ...HelmChartMutations(stores),
      ...WatchMutations(stores),
      ...UpdateMutations(stores),
      ...UnforkMutations(stores),
      ...NotificationMutations(stores),
      ...InitMutations(stores),
      ...LicenseMutations(stores),
    };
  }
  return {
    Query: query,
    Mutation: mutation,
  };
};
