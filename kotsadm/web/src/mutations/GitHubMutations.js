import gql from "graphql-tag";

export const createGithubNonce = gql`
  mutation {
    createGithubNonce
  }
`;

export const createGithubAuthToken = gql`
  mutation createGithubAuthToken($code: String!, $state: String!) {
    createGithubAuthToken(code: $code, state: $state) {
      access_token
    }
  }
`;

export const logout = gql`
  mutation {
    logout
  }
`;

