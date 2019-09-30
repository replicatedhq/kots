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
import { PrefightQueries } from "../preflight";
import { AppsQueries } from "../apps";
import { KotsQueries, KotsMutations } from "../kots_app";

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
      ...LicenseQueries(stores),
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
      ...LicenseMutations(stores),
      ...KotsMutations(stores),

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
    };
  }
  return {
    Query: query,
    Mutation: mutation,
  };
};
