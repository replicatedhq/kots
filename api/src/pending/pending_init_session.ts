export interface PendingInitSession {
  id: string;
  title: string,
  upstreamURI: string;
  requestedUpstreamURI: string;
  createdAt: Date;
  finishedAt?: Date;
  result?: string;
}
