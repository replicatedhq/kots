const UserInfo = `
type UserInfo {
  avatarUrl: String
  username: String
}`;

const GQLAccessToken = `
type AccessToken {
  access_token: String!
}`;

export default [
  UserInfo,
  GQLAccessToken
];
