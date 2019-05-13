import GitHubInstallation from "../github_installation/types";
import User from "../user/graphql/user_types";
import Cluster from "../cluster/graphql/cluster_types";
import Unfork from "../unfork/graphql/unfork_types";
import Feature from "../feature/graphql/feature_types";
import Init from "../init/graphql/init_types";
import Watch from "../watch/graphql/watch_types";
import ImageWatch from "../imagewatch/graphql/imagewatch_types";
import Update from "../update/graphql/update_types";
import Notification from "../notification/graphql/notification_types";

import { all as Mutation } from "./mutation";
import { Healthz, Query } from "./query";

export const SchemaDefinition = `
schema {
  query: Query
  mutation: Mutation
}
`;

export const ShipClusterSchemaTypes = [
  SchemaDefinition,
  Query,
  ...Mutation,
  Healthz,
  ...User,
  ...GitHubInstallation,
  ...Watch,
  ...Cluster,
  ...Feature,
  ...Notification,
  ...Init,
  ...Unfork,
  ...Update,
  ...ImageWatch,
];
