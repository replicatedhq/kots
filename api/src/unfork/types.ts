const UnforkSession = `
type UnforkSession {
  id: ID
  upstreamUri: String
  forkUri: String
  createdOn: String
  finishedOn: String
  result: String
}`;

export const types = [UnforkSession];
