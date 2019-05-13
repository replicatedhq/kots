const ImageWatch = `
type ImageWatch {
  id: ID
  name: String
  lastCheckedOn: String
  isPrivate: Boolean
  versionDetected: String
  latestVersion: String
  compatibleVersion: String
  versionsBehind: Int
  path: String
}
`;

export default [
  ImageWatch
];
