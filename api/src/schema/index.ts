import { makeExecutableSchema } from "graphql-tools";
import { Service } from "ts-express-decorators";
import { ShipClusterSchemaTypes } from "./schemaTypes";
import { Resolvers } from "./resolvers";

@Service()
export class ShipClusterSchema {
  getSchema(stores): {} {
    return makeExecutableSchema({
      typeDefs: ShipClusterSchemaTypes,
      ...Resolvers(stores),
    });
  }
}
