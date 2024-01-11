import Loader from "../shared/Loader";
import "@src/scss/components/AirgapUploadProgress.scss";

let percentage;

function moveBar(progressData) {
  const elem = document.getElementById("supportBundleStatusBar");
  const calcPercent =
    (progressData.collectorsCompleted / progressData.collectorCount) * 100;
  percentage = calcPercent > 98 ? 98 : calcPercent.toFixed();
  if (elem) {
    elem.style.width = percentage + "%";
  }
}

export default function SupportBundleCollectProgress(props) {
  const { progressData } = props;

  let progressBar;

  if (progressData.collectorsCompleted > 0) {
    moveBar(progressData);
    progressBar = (
      <div className="progressbar">
        <div
          className="progressbar-meter"
          id="supportBundleStatusBar"
          style={{ width: "0px" }}
        />
      </div>
    );
  } else {
    percentage = "0";
    progressBar = (
      <div className="progressbar">
        <div
          className="progressbar-meter"
          id="supportBundleStatusBar"
          style={{ width: "0px" }}
        />
      </div>
    );
  }

  let statusDiv = (
    <div className="u-marginTop--20 u-fontWeight--medium u-lineHeight--medium u-textAlign--center">
      <div className="flex flex1 u-marginBottom--10 justifyContent--center alignItems--center u-textColor--secondary">
        {progressData?.message && (
          <Loader className="flex u-marginRight--5" size="24" />
        )}
        {percentage >= 98 ? (
          <p>Almost done, finalizing your bundle...</p>
        ) : (
          <p>Analyzing {progressData?.message}</p>
        )}
      </div>
    </div>
  );

  return (
    <div className="PreflightProgress--wrapper flex-1-auto flex-column alignItems--center justifyContent--center u-marginTop--10">
      <div className="flex1 flex-column u-textColor--primary">
        <div className="flex1 flex-column alignItems--center justifyContent--center">
          <h1 className="u-fontSize--larger u-fontWeight--bold u-marginBottom--10">
            Analyzing {props.appTitle}
          </h1>
          <div className="flex alignItems--center u-marginTop--20">
            <span className="u-fontWeight--bold u-fontSize--normal u-textColor--secondary u-marginRight--10">
              {percentage + "%"}
            </span>
            {progressBar}
            <span className="u-fontWeight--bold u-fontSize--normal u-textColor--secondary u-marginRight--10">
              100%
            </span>
          </div>
          {statusDiv}
        </div>
      </div>
    </div>
  );
}
