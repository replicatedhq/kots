import React from "react"
import { withRouter } from "react-router-dom";
import { compose, withApollo, graphql} from "react-apollo";
import { getAirgapInstallStatus } from "../queries/AppsQueries";
import { formatByteSize } from "@src/utilities/utilities";
import Loader from "@src/components/shared/Loader";
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
  let percentage;
  // Denotes if the upload is complete (not processing)
  const isComplete = total === sent && total > 0;

  if (total > 0 && sent > 0) {
    percentage = Math.floor((sent / total) * 100).toFixed() + "%";
    progressBar =
      <div className="progressbar">
        <div className="progressbar-meter" style={{ width: `${(sent / total) * 600}px` }} />
      </div>
  } else {
    percentage = "0%";
    progressBar = (
      <div className="progressbar">
        <div className="progressbar-meter" style={{ width: "0px" }} />
      </div>
    );
  }

  if (isComplete) {
    props.data?.startPolling(2000);
  }

  return (
    <div className="AirgapUploadProgress--wrapper flex1 flex-column alignItems--center justifyContent--center">
      <div className="flex1 flex-column alignItems--center justifyContent--center u-color--tuna">
        {isComplete && <Loader size="60" color="#326DE6" />}
        <h1 className="u-fontSize--larger u-fontWeight--bold u-marginBottom--10">
          {isComplete ? "Processing" : "Uploading"} your airgap bundle
        </h1>
        {!isComplete && (
          <div className="flex alignItems--center">
            <span>{percentage}</span>
            {progressBar}
            <span>{formatByteSize(total)}</span>
          </div>
        )}
        <div className="u-marginTop--20 u-color--dustyGray u-fontWeight--bold u-lineHeight--medium u-textAlign--center">
          {getAirgapInstallStatus?.currentMessage} <br/>
          This may take a while depending on your network connection and size of your bundle
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
