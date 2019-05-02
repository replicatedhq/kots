export interface SessionModel {
  id: string;
  user_id: string;
  metadata?: string;
  expiry: Date;
}
