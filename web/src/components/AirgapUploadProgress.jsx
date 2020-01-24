import React from "react"
import { withRouter } from "react-router-dom";
import { compose, withApollo, graphql} from "react-apollo";
import { getAirgapInstallStatus } from "../queries/AppsQueries";
import { formatByteSize } from "@src/utilities/utilities";
import "@src/scss/components/AirgapUploadProgress.scss";

function AirgapUploadProgress(props) {
  const { total, sent, onProgressError, onProgressSuccess, smallSize } = props;
  const { getAirgapInstallStatus } = props.data;

  if (getAirgapInstallStatus?.installStatus === "installed") {
    // this conditional is really awkward but im keeping the functionality the same
    if (!smallSize) {
      props.data?.stopPolling();
    }
    if (onProgressSuccess) {
      onProgressSuccess();
    }
    if (!smallSize) {
      return null;
    }
  }

  const hasError = getAirgapInstallStatus?.installStatus === "airgap_upload_error";

  if (hasError) {
    props.data?.stopPolling();
    onProgressError(getAirgapInstallStatus?.currentMessage);
    return null;
  }

  let progressBar;
  let percentage;

  if (total > 0 && sent > 0) {
    percentage = Math.floor((sent / total) * 100).toFixed() + "%";
    progressBar = (
      <div className={`progressbar ${smallSize && "small"}`}>
        <div className="progressbar-meter" style={{ width: `${(sent / total) * (smallSize ? 100 : 600)}px` }} />
      </div>
    );
  } else {
    percentage = "0%";
    progressBar = (
      <div className={`progressbar ${smallSize && "small"}`}>
        <div className="progressbar-meter" style={{ width: "0px" }} />
      </div>
    );
  }

  props.data?.startPolling(2000);

  const statusMsg = getAirgapInstallStatus?.currentMessage;

  let statusDiv = (
    <div
      className={`u-marginTop--20 u-color--dustyGray u-fontWeight--bold u-lineHeight--medium u-textAlign--center`}
    >
      {statusMsg} <br/>
      This may take a while depending on your network connection and size of your bundle
    </div>
  );
  
  if (smallSize) {
    statusDiv = statusMsg && (
      <div
        className={`u-marginTop--10 u-paddingRight--30 u-color--dustyGray u-fontWeight--bold u-lineHeight--medium u-textAlign--center`}
        style={{ maxWidth: 200 }}
      >
        {statusMsg.substring(0, 30) + "..."}
      </div>
    );
  }

  return (
    <div className="AirgapUploadProgress--wrapper flex1 flex-column alignItems--center justifyContent--center">
      <div className="flex1 flex-column alignItems--center justifyContent--center u-color--tuna">
        <h1 className={`${smallSize ? "u-fontSize--large" : "u-fontSize--larger"} u-fontWeight--bold u-marginBottom--10`}>
          Uploading your airgap bundle
        </h1>
        <div className="flex alignItems--center">
          <span>{percentage}</span>
          {progressBar}
          <span>{formatByteSize(total)}</span>
        </div>
        {statusDiv}
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
