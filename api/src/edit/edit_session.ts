export interface EditSession {
  id: string;
  watchId: string,
  createdOn: Date;
  finishedOn?: Date;
  result?: string;
  isHeadless: boolean;
}
