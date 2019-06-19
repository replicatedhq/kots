import React from "react";
// import "../../scss/components/watches/WatchDetailPage.scss";

export default function WatchLicense(/*props*/) {

  return (
    <div className="flex justifyContent--center">
      <div className="LicenseDetails--wrapper u-textAlign--left u-paddingRight--20 u-paddingLeft--20">
        <p className="u-fontWeight--bold u-color--tuna u-fontSize--larger u-marginBottom--20 u-paddingBottom--5 u-lineHeight--normal">License details</p>
        <div className="u-color--tundora u-fontSize--normal u-fontWeight--medium">
          <div className="flex u-marginBottom--20">
            <p className="u-marginRight--10">Assigned release channel:</p>
            <p className="u-fontWeight--bold u-color--tuna">Title</p>
          </div>
          <div className="flex u-marginBottom--20">
            <p className="u-marginRight--10">Created:</p>
            <p className="u-fontWeight--bold u-color--tuna">May 15, 2019</p>
          </div>
          <div className="flex u-marginBottom--20">
            <p className="u-marginRight--10">Expires:</p>
            <p className="u-fontWeight--bold u-color--tuna">Never</p>
          </div>
          <div className="flex u-marginBottom--20">
            <p className="u-marginRight--10">License Type:</p>
            <p className="u-fontWeight--bold u-color--tuna">Development only</p>
          </div>
          <div className="flex u-marginBottom--20">
            <p className="u-marginRight--10">Number of Seats:</p>
            <p className="u-fontWeight--bold u-color--tuna">100</p>
          </div>
          <div className="flex u-marginBottom--20">
            <p className="u-marginRight--10">Server Host Port:</p>
            <p className="u-fontWeight--bold u-color--tuna">0.0.0.0</p>
          </div>
          <div className="flex u-marginBottom--20">
            <p className="u-marginRight--10">Server Port:</p>
            <p className="u-fontWeight--bold u-color--tuna">35600</p>
          </div>
          <div className="flex u-marginBottom--20">
            <p className="u-marginRight--10">Airgap Enabled:</p>
            <p className="u-fontWeight--bold u-color--tuna">No</p>
          </div>
        </div>
      </div>
    </div>
  );
}
