import { UserQueries, UserMutations } from "../user";
import { ClusterMutations, ClusterQueries } from "../cluster";
import { WatchMutations, WatchQueries } from "../watch";
import { UpdateMutations, UpdateQueries } from "../update";
import { UnforkQueries, UnforkMutations } from "../unfork";
import { NotificationQueries, NotificationMutations } from "../notification";
import { InitMutations, InitQueries } from "../init";
import { FeatureMutations, FeatureQueries } from "../feature";

export const Resolvers = (stores: any) => ({
  Query: {
    ...UserQueries,
    ...ClusterQueries(stores),
    ...WatchQueries(stores),
    ...UpdateQueries(stores),
    ...UnforkQueries(stores),
    ...NotificationQueries(stores),
    ...InitQueries(stores),
    ...FeatureQueries(stores),
  },

  Mutation: {
    ...UserMutations(stores),
    ...ClusterMutations(stores),
    ...WatchMutations(stores),
    ...UpdateMutations(stores),
    ...UnforkMutations(stores),
    ...NotificationMutations(stores),
    ...InitMutations(stores),
    ...FeatureMutations(stores),
  }
})
