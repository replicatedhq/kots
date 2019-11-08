import { KEntitlement } from './';

export class KLicense {
  public id: string;
  public expiresAt: string;
  public channelName: string;
  public licenseSequence?: number;
  public entitlements?: Array<KEntitlement>;
}
