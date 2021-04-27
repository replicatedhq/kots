import React from "react"
import Loader from "../shared/Loader";
import "@src/scss/components/AirgapUploadProgress.scss";
import { getReadableCollectorName } from "../../utilities/utilities";

let percentage;

function moveBar(count) {
  const elem = document.getElementById("preflighStatusBar");
  percentage = count > 21 ? 96 : (count * 4.5).toFixed();
  if (elem) {
    elem.style.width = percentage + "%";
  }
}

export default function PreflightProgress(props) {
  const { progressData, preflightResultCheckCount } = props;

  let progressBar;

  if (preflightResultCheckCount > 0) {
    moveBar(preflightResultCheckCount);
    progressBar = (
      <div className="progressbar">
        <div className="progressbar-meter" id="preflighStatusBar" />
      </div>
    );
  } else {
    percentage = "0%";
    progressBar = (
      <div className="progressbar">
        <div className="progressbar-meter" id="preflighStatusBar" style={{ width: "0px" }} />
      </div>
    );
  }
  
  const readableName = getReadableCollectorName(progressData?.currentName);
  let statusDiv = (
    <div
      className={`u-marginTop--20 u-fontWeight--medium u-lineHeight--medium u-textAlign--center`}
    >
      <div className="flex flex1 u-marginBottom--10 justifyContent--center alignItems--center u-textColor--secondary">
        {progressData?.currentName && <Loader className="flex u-marginRight--5" size="24" />}
        <p>{readableName}</p>
      </div>
    </div>
  );

  return (
    <div className="PreflightProgress--wrapper flex-1-auto flex-column alignItems--center justifyContent--center u-marginTop--10">
      <div className="flex1 flex-column u-textColor--primary">
          <div className="flex1 flex-column alignItems--center justifyContent--center">
            <h1 className="u-fontSize--larger u-fontWeight--bold u-marginBottom--10">Collecting information about your cluster</h1>
            <div className="flex alignItems--center u-marginTop--20">
              <span className="u-fontWeight--bold u-fontSize--normal u-textColor--secondary u-marginRight--10">{percentage + "%"}</span>
              {progressBar}
              <span className="u-fontWeight--bold u-fontSize--normal u-textColor--secondary u-marginRight--10">100%</span>
            </div>
            {statusDiv}
          </div>
      </div>
    </div>
  );
}
