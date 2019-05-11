import User from "../user/graphql/user_types";
import { types as GithubTypes, vendor as GithubSchema } from "../github_installation/types";
import { types as ImageWatchTypes } from "../imagewatch/types";
import { types as FeatureTypes } from "../feature/types";
import { types as InitTypes } from "../init/types";
import { types as UnforkTypes } from "../unfork/types";
import { types as NotificationTypes } from "../notification/types";
import { types as UpdateTypes } from "../update/types";
import { types as WatchTypes } from "../watch/types";
import Cluster from "../cluster/graphql/cluster_types";
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
  ...GithubTypes,
  ...GithubSchema,
  ...WatchTypes,
  ...Cluster,
  ...FeatureTypes,
  ...NotificationTypes,
  ...InitTypes,
  ...UnforkTypes,
  ...UpdateTypes,
  ...ImageWatchTypes,
];
