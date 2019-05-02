import { makeExecutableSchema } from "graphql-tools";
import { Service } from "ts-express-decorators";
import { DecoratedMutations, DecoratedQueries } from "./decorators";
import { VendorSchemaTypes } from "./schemaTypes";

@Service()
export class ReplicatedSchema {
  getSchema(): {} {
    return makeExecutableSchema({
      typeDefs: VendorSchemaTypes,
      resolvers: {
        Query: {
          ...DecoratedQueries("ship-cloud"),
        },
        Mutation: {
          ...DecoratedMutations("ship-cloud"),
        },
      },
    });
  }
}
