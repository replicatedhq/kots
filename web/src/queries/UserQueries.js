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

export const isSecuredRaw = `
  query isSecured {
    isSecured
  }
`;
export const isSecured = gql(isSecuredRaw);
