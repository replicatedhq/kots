export interface InitSession {
  id: string;
  upstreamURI: string;
  createdOn: Date;
  finishedOn?: Date;
  result: string;
}
