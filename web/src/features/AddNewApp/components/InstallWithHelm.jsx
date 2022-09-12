import React from "react";

function InstallWithHelm() {
  return (
    <div
      className={`UploadLicenseFile--wrapper flex-column flex1 u-overflow--auto Login-wrapper justifyContent--center alignItems--center`}
    >
      <div className="LoginBox-wrapper u-flexTabletReflow  u-flexTabletReflow flex-auto">
        <div className="flex-auto flex-column login-form-wrapper secure-console justifyContent--center">
          <div className="flex-column alignItems--center">
            <span className="icon helm" style={{ zoom: 5 }} />
            <div className="flex flex-column">
              <p className="u-fontSize--header2 u-textColor--primary u-fontWeight--bold u-textAlign--center u-marginTop--10 u-paddingTop--5">
                Install a Helm chart
              </p>
            </div>
          </div>
        </div>
      </div>
    </div >
  )
}

export { InstallWithHelm };