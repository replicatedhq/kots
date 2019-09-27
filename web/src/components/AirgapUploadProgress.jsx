import React from "react"
import classNames from "classnames";
import { withRouter } from "react-router-dom";
import { compose, withApollo, graphql} from "react-apollo";
import { getAirgapInstallStatus } from "../queries/AppsQueries";
import Loader from "./shared/Loader";

import "@src/scss/components/AirgapUploadProgress.scss";

function AirgapUploadProgress(props) {
  const { history, total, sent, onProgressError } = props;
  const { getAirgapInstallStatus } = props.data;

  if (getAirgapInstallStatus?.installStatus === "installed") {
    history.replace("/");
  }
  const hasError = getAirgapInstallStatus?.installStatus === "airgap_upload_error";

  if (hasError) {
    props.data?.stopPolling();
    onProgressError(getAirgapInstallStatus?.currentMessage);
    return null;
  }

  let progressBar;
  if (total > 0 && sent > 0) {
    progressBar =
      <div className="progressbar u-marginBottom--10">
        <div className="progressbar-meter" style={{ width: `${(sent / total) * 500}px` }} />
      </div>
  }

  if (total === sent && total > 0) {
    props.data?.startPolling(2000);
    progressBar = null;
  }

  return (
    <div className="AirgapUploadProgress--wrapper flex1 flex-column alignItems--center justifyContent--center">
      <div className="flex1 flex-column alignItems--center justifyContent--center u-color--tuna">
        {progressBar || <Loader size={60} color="#326DE6" />}
        {progressBar ? "Uploading" : "Processing"} your airgap bundle<br />
        <div className={classNames("u-marginTop--20", {
          "u-color--chestnut": hasError
        })}>
          {!progressBar && getAirgapInstallStatus?.currentMessage}
        </div>
      </div>
    </div>
  );
}

AirgapUploadProgress.defaultProps = {
  total: 0,
  sent: 0
};

export default compose(
  withRouter,
  withApollo,
  graphql(getAirgapInstallStatus, {
    options: () => {
      return {
        fetchPolicy: "network-only"
      };
    }
  })
)(AirgapUploadProgress);
