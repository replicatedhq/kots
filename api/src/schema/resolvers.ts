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
import { ImageWatchMutations, ImageWatchQueries } from "../imagewatch";
import { HealthzQueries } from "../healthz";
import { Params } from "../server/params";
import { Stores } from "./stores";
import { GithubInstallationQueries, GithubInstallationMutations } from "../github_installation";
import { HelmChartQueries, HelmChartMutations } from "../helmchart";
import { TroubleshootQueries, TroubleshootMutations } from "../troubleshoot";
import { LicenseQueries, LicenseMutations } from "../license";

export const Resolvers = (stores: Stores, params: Params) => ({
  Query: {
    ...UserQueries(stores),
    ...ClusterQueries(stores),
    ...WatchQueries(stores),
    ...UpdateQueries(stores),
    ...UnforkQueries(stores),
    ...NotificationQueries(stores),
    ...InitQueries(stores),
    ...FeatureQueries(stores),
    ...HealthzQueries(stores),
    ...GithubInstallationQueries(stores),
    ...EditQueries(stores),
    ...PendingQueries(stores),
    ...HelmChartQueries(stores),
    ...ImageWatchQueries(stores),
    ...TroubleshootQueries(stores),
    ...LicenseQueries(stores),
  },

  Mutation: {
    ...UserMutations(stores, params),
    ...ClusterMutations(stores),
    ...WatchMutations(stores),
    ...UpdateMutations(stores),
    ...UnforkMutations(stores),
    ...NotificationMutations(stores),
    ...InitMutations(stores),
    ...FeatureMutations(stores),
    ...GithubInstallationMutations(stores),
    ...EditMutations(stores),
    ...HelmChartMutations(stores),
    ...ImageWatchMutations(stores),
    ...TroubleshootMutations(stores),
    ...LicenseMutations(stores),
  }
})
