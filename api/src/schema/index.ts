import { makeExecutableSchema } from "graphql-tools";
import { ShipClusterSchemaTypes } from "./schemaTypes";
import { Resolvers } from "./resolvers";
import { Stores } from "./stores";
import { Params } from "../server/params";

export class ShipClusterSchema {
  getSchema(stores: Stores, params: Params): {} {
    return makeExecutableSchema({
      typeDefs: ShipClusterSchemaTypes,
      resolvers: Resolvers(stores, params),
    });
  }
}
