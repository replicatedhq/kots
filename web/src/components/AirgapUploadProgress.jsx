import React from "react"
import { withRouter } from "react-router-dom";
import { compose, withApollo, graphql } from "react-apollo";
import Loader from "./shared/Loader";
import { getAirgapInstallStatus } from "../queries/AppsQueries";
import { formatByteSize, calculateTimeDifference } from "@src/utilities/utilities";
import "@src/scss/components/AirgapUploadProgress.scss";
import get from "lodash/get";
let processingImages = null;

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

  if (props.unkownProgress) {
    return (
      <div>
        <Loader className="flex justifyContent--center" size="32" />
        <p className="u-marginTop--10 u-fontSize--small u-color--dustyGray u-fontWeight--medium" style={{ maxWidth: 200 }}>
          This may take a while depending on your network connection and size of your bundle
         </p>
      </div>
    )
  }

  let progressBar;
  let percentage;
  let uploadComplete;

  if (total > 0 && sent > 0) {
    uploadComplete = sent === total
    percentage = Math.floor((sent / total) * 100).toFixed() + "%";
    progressBar = (
      <div className={`progressbar ${smallSize ? "small" : ""}`}>
        <div className={`progressbar-meter ${uploadComplete ? "complete" : ""}`} style={{ width: `${(sent / total) * (smallSize ? 100 : 600)}px` }} />
      </div>
    );
  } else {
    percentage = "0%";
    progressBar = (
      <div className={`progressbar ${smallSize ? "small" : ""}`}>
        <div className="progressbar-meter" style={{ width: "0px" }} />
      </div>
    );
  }

  props.data?.startPolling(1000);
  
  let statusMsg = getAirgapInstallStatus?.currentMessage;
  try {
    // Some of these messages will be JSON formatted progress reports.
    let jsonMessage;
    jsonMessage = JSON.parse(statusMsg);
    const type = get(jsonMessage, "type");
    if (type === "progressReport") {
      try {
        const parsedMsg = JSON.parse(jsonMessage.compatibilityMessage);
        statusMsg = parsedMsg.compatibilityMessage;
        processingImages = parsedMsg.images.sort((a, b) => (a.status > b.status) ? -1 : 1);
      } catch {
        statusMsg = jsonMessage.compatibilityMessage;
      }
    }
  } catch {
    // empty
  }

  let statusDiv = (
    <div
      className={`u-marginTop--20 u-color--dustyGray u-fontWeight--medium u-lineHeight--medium u-textAlign--center`}
    >
      <p className="u-marginBottom--5">{statusMsg}</p>
      <p>This may take a while depending on your network connection and size of your bundle</p>
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
      <div className="flex1 flex-column u-color--tuna">
        {processingImages ?
          <div className="flex1 flex-column alignItems--center justifyContent--center">
            <div className="flex-auto">
              <h1 className="u-fontSize--larger u-fontWeight--bold u-marginBottom--10 u-textAlign--center">
                Pushing {processingImages?.length} image{processingImages?.length === 1 ? "" : "s"} to your registry
              </h1>
              {processingImages?.map((image, i) => {
                let imageProgressBar;
                let percentage;
              
                if (image.total > 0 && image.current > 0) {
                  percentage = Math.floor((image.current / image.total) * 100).toFixed() + "%";
                  imageProgressBar = (
                    <div className="progressbar">
                      <div className={`progressbar-meter ${image.status === "uploaded" ? "complete" : ""}`} style={{ width: `${(image.current / image.total) * (600)}px` }} />
                    </div>
                  );
                } else {
                  percentage = "0%";
                  imageProgressBar = (
                    <div className="progressbar u-opacity--half">
                      <div className={`progressbar-meter ${image.status === "uploaded" ? "complete" : ""}`} style={{ width: "0px" }} />
                    </div>
                  );
                }
                let currentMessage = "Waiting to start";
                if (image.error !== "") {
                  currentMessage = image.error;
                } else if (image.status === "uploaded") {
                  const completedTime = calculateTimeDifference(image.startTime, image.endTime);
                  currentMessage = `Completed in ${completedTime}`;
                } else if (image.status === "uploading") {
                  currentMessage = statusMsg;
                }

                return (
                  <div key={`${image.displayName}-${i}`} className="flex1 u-marginTop--20">
                    <div className="flex flex1 alignItems--center">
                      <p className={`u-fontWeight--bold u-fontSize--normal u-color--tundora u-marginRight--10 u-textAlign--right flex1 ${image.status === "queued" ? "u-opacity--half" : ""}`}>{image.displayName}</p>
                      {imageProgressBar}
                      {image.status === "uploaded" ? <span className="u-marginLeft--10 icon checkmark-icon" /> : <span className="u-fontWeight--medium u-fontSize--normal u-color--tundora u-marginLeft--10">{percentage}</span>}
                    </div>
                    <div className="u-marginTop--5">
                      {currentMessage ? <p className="u-textAlign--center u-fontSize--small u-fontWeight--medium u-color--dustyGray">{currentMessage}</p> : <p className="u-fontSize--small"></p>}
                    </div>
                  </div>
                )
              })}
            </div>
          </div>
        :
          <div className="flex1 flex-column alignItems--center justifyContent--center">
            <h1 className={`${smallSize ? "u-fontSize--large" : "u-fontSize--larger"} u-fontWeight--bold u-marginBottom--10`}>
              Uploading your airgap bundle
            </h1>
            <div className="flex alignItems--center u-marginTop--20">
              <span className="u-fontWeight--bold u-fontSize--normal u-color--tundora u-marginRight--10">{percentage}</span>
              {progressBar}
              {uploadComplete ? <span className="u-marginLeft--10 icon checkmark-icon" /> : <span className="u-fontWeight--medium u-fontSize--normal u-color--tundora u-marginLeft--10">{formatByteSize(total)}</span>}
            </div>
            {statusDiv}
          </div>
        }
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
