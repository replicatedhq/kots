import { UserQueries, UserMutations } from "../user";
import { ClusterMutations, ClusterQueries } from "../cluster";
import { WatchMutations, WatchQueries } from "../watch";
import { UpdateMutations, UpdateQueries } from "../update";
import { UnforkQueries, UnforkMutations } from "../unfork";
import { NotificationQueries, NotificationMutations } from "../notification";
import { InitMutations, InitQueries } from "../init";
import { FeatureMutations, FeatureQueries } from "../feature";
import { HealthzQueries } from "../healthz";

import { Params } from "../server/params";
import { Stores } from "./stores";
import { GithubInstallationQueries, GithubInstallationMutations } from "../github_installation";

export const Resolvers = (stores: Stores, params: Params) => ({
  Query: {
    ...UserQueries,
    ...ClusterQueries(stores),
    ...WatchQueries(stores),
    ...UpdateQueries(stores),
    ...UnforkQueries(stores),
    ...NotificationQueries(stores),
    ...InitQueries(stores),
    ...FeatureQueries(stores),
    ...HealthzQueries(stores),
    ...GithubInstallationQueries(stores),
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
  }
})
