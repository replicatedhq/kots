export interface UpdateSession {
  id: string;
  watchId: string;
  userId: string;
  createdOn: string;
  finishedOn?: string;
  result?: string;
}
