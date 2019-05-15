export interface UpdateSession {
  id: string;
  watchId: string;
  createdOn: Date;
  finishedOn?: Date;
  result?: string;
}
