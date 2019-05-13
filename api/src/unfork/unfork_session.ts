export interface UnforkSession {
  id: string;
  upstreamURI: string;
  forkURI: string;
  createdOn: Date;
  finishedOn?: Date;
  result?: string;
}