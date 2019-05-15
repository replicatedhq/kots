import gql from "graphql-tag";

export const userInfo = gql`
  query {
    userInfo {
      username,
      avatarUrl
    }
  }
`;
