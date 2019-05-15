const UpdateSession = `
type UpdateSession {
  id: ID
  watchId: ID
  userId: ID
  createdOn: String
  finishedOn: String
  result: String
}`;

export default [
 UpdateSession,
];
