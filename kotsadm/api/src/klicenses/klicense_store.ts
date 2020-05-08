import pg from "pg";
import request from "request-promise";
import { Params } from "../server/params";
import { ReplicatedError } from "../server/errors";
import { KLicense } from "./";
import { getLicenseInfoFromYaml } from "../util/utilities";

export class KotsLicenseStore {
  constructor(
    private readonly pool: pg.Pool,
    private readonly params: Params,
  ) {
  }

  public async getAppLicense(appId: string): Promise<KLicense> {
    try {
      // Get current app license
      let q = `select kots_license as license from app_version where app_id = $1 order by sequence desc limit 1`;
      let v: any[] = [appId];

      let result = await this.pool.query(q, v);
      if (result.rows.length === 0 || !result.rows[0].license) {
        q = `select license from app where id = $1`;
        v = [appId];
        result = await this.pool.query(q, v);
        if (result.rows.length === 0) {
          throw new ReplicatedError(`No license found for app with an ID of ${appId}`);
        }
      }

      return getLicenseInfoFromYaml(result.rows[0].license);
    } catch (err) {
      throw new ReplicatedError(`Failed to get app license ${err}`);
    }
  }

  public async getAppLicenseSpec(appId: string): Promise<string | undefined> {
    const q = `select kots_license as license from app_version where app_id = $1 order by sequence desc limit 1`;
    const v = [
      appId,
    ];

    const result = await this.pool.query(q, v);

    if (result.rows.length === 0) {
      throw new ReplicatedError(`No license found for app with an ID of ${appId}`);
    }

    return result.rows[0].license;
  }
}
