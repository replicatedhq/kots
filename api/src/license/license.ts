import { Entitlement } from './';
import * as _ from "lodash";


export class License {
  public id: string;
  public channel: string;
  public createdAt: string;
  public expiresAt: string;
  public type: string;
  public entitlements?: Array<Entitlement>;
}
