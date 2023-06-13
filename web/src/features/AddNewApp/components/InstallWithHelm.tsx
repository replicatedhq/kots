import { useApps } from "@features/App";
import React, { useEffect } from "react";
import { useNavigate } from "react-router-dom";

function InstallWithHelm() {
  // TODO: // move this into a routes component
  // poll for apps data and redirect if app is installed
  const { apps } = useApps({ refetchInterval: 2000 }).data || {};
  const navigate = useNavigate();

  useEffect(() => {
    if (apps && apps?.length > 0) {
      navigate("/apps");
    }
  }, [apps?.length]);

  return (
    <div
      className={`UploadLicenseFile--wrapper flex-column flex1 u-overflow--auto Login-wrapper justifyContent--center alignItems--center u-textAlign--center`}
    >
      <div className="LoginBox-wrapper u-flexTabletReflow  u-flexTabletReflow flex-auto">
        <div className="flex-auto flex-column login-form-wrapper secure-console justifyContent--center">
          <div className="flex-column alignItems--center">
            <span className="icon helm" style={{ zoom: 3 }} />
            <div className="flex flex-column">
              <p className="u-fontSize--header2 u-textColor--secondary u-fontWeight--bold u-textAlign--center u-marginTop--10 u-paddingTop--5">
                Install a Helm chart
              </p>
              <p
                className="u-fontSize--normal u-textColor--accent u-fontWeight--medium u-lineHeight--normal u-marginTop--20 u-marginRight--30 u-marginLeft--30"
                style={{ maxWidth: "300px" }}
              >
                In order to use the admin console you need to install a Helm
                chart.
              </p>
            </div>
          </div>
        </div>
      </div>
      <p className="u-fontSize--small u-textColor--accent u-fontWeight--normal u-lineHeight--normal u-marginTop--10">
        To learn more,{" "}
        <a
          href="https://docs.replicated.com/vendor/helm-install"
          target="_blank"
        >
          read the documentation
        </a>{" "}
        on supporting Helm CLI installations.
      </p>
    </div>
  );
}

export { InstallWithHelm };
