import React from "react"
import { withRouter } from "react-router-dom";
import { compose, withApollo, graphql} from "react-apollo";
import { getOnlineInstallStatus } from "../queries/AppsQueries";
import "@src/scss/components/AirgapUploadProgress.scss";

function LicenseUploadProgress(props) {
  const { getOnlineInstallStatus } = props.data;

  props.data?.startPolling(2000);

  const statusMsg = getOnlineInstallStatus?.currentMessage;

  let statusDiv = (
    <div className={`u-marginTop--20 u-lineHeight--medium u-textAlign--center`}>
      <p className="u-color--tundora u-fontSize--normal u-fontWeight--bold u-marginBottom--10 u-paddingBottom--5">{statusMsg}</p>
      <p className="u-fontSize--small u-color--dustyGray u-fontWeight--medium">This may take a while depending on your network connection and size of your bundle</p>
    </div>
  );

  return (
    <div className="AirgapUploadProgress--wrapper flex1 flex-column alignItems--center justifyContent--center">
      <div className="flex1 flex-column alignItems--center justifyContent--center u-color--tuna">
        <p className="u-marginTop--10 u-paddingTop--5 u-fontSize--header u-color--tuna u-fontWeight--bold">Installing your license</p>
        {statusDiv}
      </div>
    </div>
  );
}

export default compose(
  withRouter,
  withApollo,
  graphql(getOnlineInstallStatus, {
    options: () => {
      return {
        fetchPolicy: "network-only"
      };
    }
  })
)(LicenseUploadProgress);
