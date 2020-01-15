import Express from "express";
import {
  BodyParams,
  Controller,
  Get,
  Req,
  Res,
  Any,
} from "@tsed/common";
import yaml from "js-yaml";
import * as _ from "lodash";

@Controller("/license/v1")
export class LicenseAPI {
  @Get("/license")
  public async license(
    @Res() response: Express.Response,
    @Req() request: Express.Request,
    @BodyParams("") body: any,
  ): Promise<any> {
    const apps = await request.app.locals.stores.kotsAppStore.listInstalledKotsApps();
    if (_.size(apps) === 0) {
      response.status(404);
      return {};
    }

    if (_.size(apps) > 1) {
      response.status(400);
      return {};
    }

    const app = apps[0];
    const licenseYaml = await request.app.locals.stores.kotsLicenseStore.getAppLicenseSpec(app.id);

    const license = yaml.safeLoad(licenseYaml);

    const platformLicense = {
      license_id: license.spec.licenseID,
      installation_id: app.id,
      assignee: license.spec.customerName,
      release_channel: license.spec.channelName,
      license_type: license.spec.licenseType,
    }

    let fields: any[] = [];
    if (license.spec.entitlements) {
      const keys = Object.keys(license.spec.entitlements);
      for (let k = 0; k < keys.length; k++) {
        const key = keys[k];
        const entitlement = license.spec.entitlements[key];
        if (key === "expires_at") {
          if (license.spec.entitlements.expires_at.value) {
            platformLicense["expiration_time"] = license.spec.entitlements.expires_at.value;
          }
        } else {
          fields.push({
            field: key,
            title: entitlement.title,
            type: entitlement.valueType,
            value: entitlement.value,
            hide_from_customer: entitlement.isHidden,
          });
        }
      }
    }
    platformLicense["fields"] = fields;
  
    response.status(200);
    return platformLicense;
  }
}
