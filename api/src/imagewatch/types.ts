const ImageWatchItem = `
type ImageWatchItem {
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

export const types = [ImageWatchItem];
