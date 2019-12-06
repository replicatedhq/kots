import { UserQueries, UserMutations } from "../user";
import { ClusterMutations, ClusterQueries } from "../cluster";
import { FeatureMutations, FeatureQueries } from "../feature";
import { HealthzQueries } from "../healthz";
import { Params } from "../server/params";
import { Stores } from "./stores";
import { TroubleshootQueries, TroubleshootMutations } from "../troubleshoot";
import { KotsLicenseQueries, KotsLicenseMutations } from "../klicenses";
import { PrefightQueries } from "../preflight";
import { AppsQueries, AppsMutations } from "../apps";
import { KotsQueries, KotsDashboardQueries, KotsMutations } from "../kots_app";
import { KurlQueries, KurlMutations } from "../kurl";
import { MonitoringQueries, MonitoringMutations } from "../monitoring";

export const Resolvers = (stores: Stores, params: Params) => {
  let query = {
    ...FeatureQueries(stores),
    ...HealthzQueries(stores),
    ...TroubleshootQueries(stores),
    ...PrefightQueries(stores),
    ...AppsQueries(stores),
    ...ClusterQueries(stores),
    ...UserQueries(stores),
    ...MonitoringQueries(stores),
    ...KotsQueries(stores, params),
    ...KotsDashboardQueries(stores, params),
    ...KotsLicenseQueries(stores),
    ...KurlQueries(stores, params),
  };

  let mutation = {
    ...ClusterMutations(stores, params),
    ...FeatureMutations(stores),
    ...TroubleshootMutations(stores, params),
    ...UserMutations(stores, params),
    ...MonitoringMutations(stores),
    ...KotsLicenseMutations(stores),
    ...KotsMutations(stores),
    ...KurlMutations(stores, params),
    ...AppsMutations(stores),
  };

  return {
    Query: query,
    Mutation: mutation,
  };
};
