
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
  GQLAccessToken,
  AdminSignupInfo,
];
