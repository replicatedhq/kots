import React from "react"
import { withRouter } from "react-router-dom";
import { compose, withApollo, graphql} from "react-apollo";
import { getAirgapInstallStatus } from "../queries/AppsQueries";
import Loader from "./shared/Loader";

function AirgapUploadProgress(props) {
  const { history } = props;
  const { getAirgapInstallStatus } = props.data;

  if (getAirgapInstallStatus?.installStatus === "installed") {
    history.replace("/");
  }

  return (
    <div className="flex1 flex-column alignItems--center justifyContent--center">
      <div className="flex1 flex-column alignItems--center justifyContent--center u-color--tuna">
        <Loader size={60} color="#326DE6"/>
        Checking in on your airgap bundle right now...<br />
        <div className="u-marginTop--20">
          {getAirgapInstallStatus?.currentMessage}
        </div>
      </div>
    </div>
  );
}

export default compose(
  withRouter,
  withApollo,
  graphql(getAirgapInstallStatus, {
    options: () => {
      return {
        pollInterval: 2000
      }
    }
  })
)(AirgapUploadProgress);
