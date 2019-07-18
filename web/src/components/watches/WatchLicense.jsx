import React from "react";
import Helmet from "react-helmet";
import {
  Utilities,
  getWatchMetadata,
  getReadableLicenseType,
  getAssignedReleaseChannel
} from "@src/utilities/utilities";
import isEmpty from "lodash/isEmpty";

export default function WatchLicense(props) {
  const { watch } = props;
  const appMeta = getWatchMetadata(watch.metadata);

  const createdAt = Utilities.dateFormat(appMeta.license.createdAt, "MMM D, YYYY");
  const licenseType = getReadableLicenseType(appMeta.license.type);
  const assignedReleaseChannel = getAssignedReleaseChannel(watch.stateJSON);

  // TODO: We shuold probably return something different if it never expires to avoid this hack string check.
  let expiresAt = "";
  if (!isEmpty(appMeta)) {
    expiresAt = appMeta.license.expiresAt === "0001-01-01T00:00:00Z" ? "Never" : Utilities.dateFormat(appMeta.license.expiresAt, "MMM D, YYYY");
  }

  return (
    <div className="flex justifyContent--center">
      <Helmet>
        <title>{`${watch.watchName} License`}</title>
      </Helmet>
      <div className="LicenseDetails--wrapper u-textAlign--left u-paddingRight--20 u-paddingLeft--20">
        <p className="u-fontWeight--bold u-color--tuna u-fontSize--larger u-marginBottom--20 u-paddingBottom--5 u-lineHeight--normal">License details</p>
        <div className="u-color--tundora u-fontSize--normal u-fontWeight--medium">
          <div className="flex u-marginBottom--20">
            <p className="u-marginRight--10">Assigned release channel:</p>
            <p className="u-fontWeight--bold u-color--tuna">{assignedReleaseChannel}</p>
          </div>
          <div className="flex u-marginBottom--20">
            <p className="u-marginRight--10">Created:</p>
            <p className="u-fontWeight--bold u-color--tuna">{createdAt}</p>
          </div>
          <div className="flex u-marginBottom--20">
            <p className="u-marginRight--10">Expires:</p>
            <p className="u-fontWeight--bold u-color--tuna">{expiresAt}</p>
          </div>
          <div className="flex u-marginBottom--20">
            <p className="u-marginRight--10">License Type:</p>
            <p className="u-fontWeight--bold u-color--tuna">{licenseType}</p>
          </div>
          {watch.entitlements && watch.entitlements.map(entitlement => {
            return (
              <div key={entitlement.key} className="flex u-marginBottom--20">
                <p className="u-marginRight--10">{entitlement.name}</p>
                <p className="u-fontWeight--bold u-color--tuna">{entitlement.value}</p>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
