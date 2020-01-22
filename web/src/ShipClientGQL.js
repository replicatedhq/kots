import { ApolloClient } from "apollo-client";
import { ApolloLink } from "apollo-link";
import { createHttpLink } from "apollo-link-http";
import { RestLink } from "apollo-link-rest";
import { setContext } from "apollo-link-context";
import { onError } from "apollo-link-error";
import { InMemoryCache } from "apollo-cache-inmemory";
import { withClientState } from "apollo-link-state";
import { Utilities } from "./utilities/utilities";
import fetch from "node-fetch";

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

  if (fetcher) {
    global.Headers = fetcher.Headers;
  }

  const restLink = new RestLink({
    uri: `${restEndpoint}/v1`,
    endpoints: {
      "v1": `${restEndpoint}/v1`,
    },
    customFetch: fetcher ? fetcher : undefined,
  });

  const authLink = setContext(async (_, { headers }) => {
    return {
      headers: {
        ...headers,
        authorization: await tokenFunction(),
        "X-Replicated-Client": "kotsadm",
      },
    };
  });

  const errorLink = onError(({ graphQLErrors, networkError }) => {
    if (graphQLErrors) {
      graphQLErrors.map(({ msg, locations, path }) => {
        if (!msg) {
          console.log(`Unknown GraphQL error`);
          return;
        }
        const unauthorized = msg === "Unauthorized" || msg.includes("Unknown session type");
        if (unauthorized) {
          Utilities.logoutUser();
        } else if (msg === "Forbidden") {
          client.writeData({ data: { showUnathorizedModal: true } });
          return msg;
        } else {
          if (process.env.NODE_ENV === "development") {
            console.log(
              "[GraphQL error]:",
              "Message:", msg, "|",
              "Location:", locations, "|",
              "Path:", path
            );
          }
        }
      })
    }
    if (networkError) {
      console.log(`[Network error]: ${networkError}`);
      if (networkError.statusCode === 403 && Utilities.isLoggedIn()) {
        Utilities.logoutUser();
      }
    }
  });

  const link = ApolloLink.from([
    stateLink,
    authLink,
    restLink,
    errorLink,
    httpLink,
  ]);

  const client = new ApolloClient({
    link,
    cache: cache
  });

  return client;
}
