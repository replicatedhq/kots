import { ApolloClient } from "apollo-client";
import { ApolloLink } from "apollo-link";
import { createHttpLink } from "apollo-link-http";
import { RestLink } from "apollo-link-rest";
import { setContext } from "apollo-link-context";
import { onError } from "apollo-link-error";
import { InMemoryCache } from "apollo-cache-inmemory";
import { withClientState } from "apollo-link-state";
import { Utilities } from "./utilities/utilities";

export function ShipClientGQL(graphqlEndpoint, restEndpoint, tokenFunction, fetcher) {
  const cache = new InMemoryCache({
    addTypename: false,
  });

  const stateLink = withClientState({
    cache
  });

  const httpLink = createHttpLink({
    uri: graphqlEndpoint,
    fetch: fetcher ? fetcher : fetch,
  });

  const restLink = new RestLink({
    uri: restEndpoint,
    fetch: fetcher ? fetcher : fetch,
  });

  const authLink = setContext(async (_, { headers }) => {
    return {
      headers: {
        ...headers,
        authorization: await tokenFunction(),
        "X-Replicated-Client": "ship-cloud",
      },
    };
  });

  const errorLink = onError(({ graphQLErrors, networkError }) => {
    if (graphQLErrors) {
      graphQLErrors.map(({ message, locations, path }) => {
        if (message === "Unauthorized") {
          Utilities.logoutUser();
        } else if (message === "Forbidden") {
          client.writeData({ data: { showUnathorizedModal: true } });
          return message;
        } else {
          if(process.env.NODE_ENV === "development") {
            console.log(`[GraphQL error]: Message: ${message}, Location: ${locations}, Path: ${path}`);
          }
        }
      })
    }
    if (networkError) {
      console.log(`[Network error]: ${networkError}`);
    }
  });

  const link = ApolloLink.from([
    stateLink,
    authLink,
    errorLink,
    httpLink,
    restLink,
  ]);

  const client = new ApolloClient({
    link,
    cache: cache
  });

  return client;
}
