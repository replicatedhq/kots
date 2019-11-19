import User from "../user/graphql/user_types";
import Cluster from "../cluster/graphql/cluster_types";
import Feature from "../feature/graphql/feature_types";
import HelmChart from "../helmchart/graphql/helmchart_types";
import Troubleshoot from "../troubleshoot/graphql/troubleshoot_types";
import KLicense from "../klicenses/graphql/klicense_types";
import Preflight from "../preflight/graphql/preflight_types";
import Apps from "../apps/graphql/apps_types";
import KotsApp from "../kots_app/graphql/kots_app_types";
import Metric from "../monitoring/graphql/metric_types";
import Kurl from "../kurl/graphql/kurl_types";

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
  ...Apps,
  ...User,
  ...KotsApp,
  ...Metric,
  ...Cluster,
  ...Feature,
  ...HelmChart,
  ...Troubleshoot,
  ...KLicense,
  ...Preflight,
  ...Kurl,
];
