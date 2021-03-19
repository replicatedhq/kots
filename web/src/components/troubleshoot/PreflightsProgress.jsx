import React from "react"
import Loader from "../shared/Loader";
import "@src/scss/components/AirgapUploadProgress.scss";
import { getReadableCollectorName } from "../../utilities/utilities";

export default class PreflightProgress extends React.Component {
  
  render() {
    const { progressData } = this.props;

    let progressBar;
    let percentage;
    let uploadComplete;

    if (progressData?.completedCount > 0) {
      uploadComplete = progressData?.completedCount === progressData?.totalCount
      percentage = (progressData?.completedCount / progressData.totalCount * 100) + "%";
      progressBar = (
        <div className="progressbar">
          <div className={`progressbar-meter ${uploadComplete ? "complete" : ""}`} style={{ width: percentage }} />
        </div>
      );
    } else {
      percentage = "0%";
      progressBar = (
        <div className="progressbar">
          <div className="progressbar-meter" style={{ width: "0px" }} />
        </div>
      );
    }
    
    let readableNameToShow;
    const readableName = getReadableCollectorName(progressData?.currentName);
    if (!readableName) {
      readableNameToShow = "Gathering details about the cluster";
    } else {
      readableNameToShow = readableName;
    }
    let statusDiv = (
      <div
        className={`u-marginTop--20 u-fontWeight--medium u-lineHeight--medium u-textAlign--center`}
      >
        <div className="flex flex1 u-marginBottom--10 justifyContent--center alignItems--center u-color--tundora">
          {progressData?.currentName && <Loader className="flex u-marginRight--5" size="24" />}
          <p>{readableNameToShow}</p>
        </div>
      </div>
    );

    return (
      <div className="PreflightProgress--wrapper flex-1-auto flex-column alignItems--center justifyContent--center u-marginTop--10">
        <div className="flex1 flex-column u-color--tuna">
            <div className="flex1 flex-column alignItems--center justifyContent--center">
              <h1 className="u-fontSize--larger u-fontWeight--bold u-marginBottom--10">Collecting information about your cluster</h1>
              <div className="flex alignItems--center u-marginTop--20">
                <span className="u-fontWeight--bold u-fontSize--normal u-color--tundora u-marginRight--10">{percentage}</span>
                {progressBar}
                <span className="u-fontWeight--bold u-fontSize--normal u-color--tundora u-marginRight--10">100%</span>
              </div>
              {statusDiv}
            </div>
        </div>
      </div>
    );
  }
}
