import { useParams } from "react-router-dom";
import Loader from "../shared/Loader";
import "@src/scss/components/AirgapUploadProgress.scss";
import { KotsParams } from "@types";

function moveBar(percentage: number) {
  const elem = document.getElementById("preflightStatusBar");
  if (elem) {
    elem.style.width = percentage.toFixed() + "%";
  }
}

interface NamesObj {
  [key: string]: string;
}
function getReadableCollectorName(name: string) {
  const namesObj: NamesObj = {
    "cluster-info": "Gathering basic information about the cluster",
    "cluster-resources": "Gathering available resources in cluster",
    mysql: "Gathering information about MySQL",
    postgres: "Gathering information about PostgreSQL",
    redis: "Gathering information about Redis",
  };
  if (name in namesObj) {
    return namesObj[name];
  }

  return "Gathering details about the cluster";
}

export default function PreflightProgress(props: {
  pendingPreflightCheckName: string;
  percentage: number;
}) {
  const { pendingPreflightCheckName, percentage } = props;
  const { slug } = useParams<keyof KotsParams>() as KotsParams;
  let progressBar;

  if (percentage > 0) {
    moveBar(percentage);
    progressBar = (
      <div className="progressbar">
        <div
          className="progressbar-meter"
          id="preflightStatusBar"
          style={{ width: "0px" }}
        />
      </div>
    );
  } else {
    progressBar = (
      <div className="progressbar">
        <div
          className="progressbar-meter"
          id="preflightStatusBar"
          style={{ width: "0px" }}
        />
      </div>
    );
  }

  const readableName = getReadableCollectorName(pendingPreflightCheckName);
  let statusDiv = (
    <div className="u-marginTop--20 u-fontWeight--medium u-lineHeight--medium u-textAlign--center">
      <div className="flex flex1 u-marginBottom--10 justifyContent--center alignItems--center u-textColor--secondary">
        {pendingPreflightCheckName && (
          <Loader className="flex u-marginRight--5" size="24" />
        )}
        <p>{readableName}</p>
      </div>
    </div>
  );

  return (
    <div className="PreflightProgress--wrapper flex-1-auto flex-column alignItems--center justifyContent--center u-marginTop--10">
      <div className="flex1 flex-column u-textColor--primary">
        <div className="flex1 flex-column alignItems--center justifyContent--center">
          <h1 className="u-fontSize--larger u-fontWeight--bold u-marginBottom--10">
            Collecting information about {slug}
          </h1>
          <div className="flex alignItems--center u-marginTop--20">
            <span className="u-fontWeight--bold u-fontSize--normal u-textColor--secondary u-marginRight--10">
              {percentage > 0 ? `${percentage}%` : "0%"}
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
