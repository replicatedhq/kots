const UserInfo = `
type UserInfo {
  avatarUrl: String
  username: String
}`;

const GQLAccessToken = `
type AccessToken {
  access_token: String!
}`;

const AdminSignupInfo = `
type AdminSignupInfo {
  token: String!
  userId: String!
}`;

export default [
  UserInfo,
  GQLAccessToken,
  AdminSignupInfo,
];
