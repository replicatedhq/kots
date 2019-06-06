import React from "react";

export default function WatchLicense(/*props*/) {

  return (
    <div className="flex alignItems--center justifyContent--center">
      <div className="centered-container u-color--tuna">
        <div className="flex u-paddingBottom--30">
          <p className="u-marginRight--10">Assigned release channel:</p>
          <p className="u-fontWeight--bold">Title</p>
        </div>
        <div className="flex u-paddingBottom--30">
          <p className="u-marginRight--10">Created:</p>
          <p className="u-fontWeight--bold">May 15, 2019</p>
        </div>
        <div className="flex u-paddingBottom--30">
          <p className="u-marginRight--10">Expires:</p>
          <p className="u-fontWeight--bold">Never</p>
        </div>
        <div className="flex u-paddingBottom--30">
          <p className="u-marginRight--10">License Type:</p>
          <p className="u-fontWeight--bold">Development only</p>
        </div>
        <div className="flex u-paddingBottom--30">
          <p className="u-marginRight--10">Number of Seats:</p>
          <p className="u-fontWeight--bold">100</p>
        </div>
        <div className="flex u-paddingBottom--30">
          <p className="u-marginRight--10">Server Host Port:</p>
          <p className="u-fontWeight--bold">0.0.0.0</p>
        </div>
        <div className="flex u-paddingBottom--30">
          <p className="u-marginRight--10">Server Port:</p>
          <p className="u-fontWeight--bold">35600</p>
        </div>
        <div className="flex u-paddingBottom--30">
          <p className="u-marginRight--10">Airgap Enabled:</p>
          <p className="u-fontWeight--bold">No</p>
        </div>
      </div>
    </div>
  );
}
