import gql from "graphql-tag";

export const userInfoRaw = `
  query userInfo {
    userInfo {
      username
      avatarUrl
    }
  }
`;
export const userInfo = gql(userInfoRaw);

export const validateRegistryInfoRaw = `
  query validateRegistryInfo($slug: String, $endpoint: String, $username: String, $password: String, $org: String) {
    validateRegistryInfo(slug: $slug, endpoint: $endpoint, username: $username, password: $password, org: $org)
  }
`;
export const validateRegistryInfo = gql(validateRegistryInfoRaw);
