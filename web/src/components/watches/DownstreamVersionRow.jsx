import React from "react";
import dayjs from "dayjs";
import classNames from "classnames";
import { Link } from "react-router-dom";
import { Utilities } from "@src/utilities/utilities";
import Loader from "../shared/Loader";

export default function DownstreamVersionRow(props) {
  const { version, downstreamWatch, isKots, urlParams, handleMakeCurrent, hasPreflight, isDeploying, onReleaseNotesClick } = props;
  if (!version) { return null; }
  const gitRef = downstreamWatch?.cluster?.gitOpsRef;
  const githubLink = gitRef && `https://github.com/${gitRef.owner}/${gitRef.repo}/pull/${version.pullrequestNumber}`;
  const prPending = version.pullrequestNumber && (version.status === "opened" || version.status === "pending");

  let shipInstallnode = null;
  if (!gitRef && (version.status === "pending" || version.status === "pending_preflight") ) {
    shipInstallnode = (
      <div className="u-marginLeft--10 flex-column flex-auto flex-verticalCenter">
        <button className="btn secondary small" onClick={() => handleMakeCurrent(urlParams.slug, version.sequence, downstreamWatch.cluster.slug, version.status)}>Make current version</button>
      </div>
    )
  }
  let deployedAtTextNode;
  if (version.deployedAt) {
    deployedAtTextNode = <span className="gh-version-detail-text">{gitRef ? "Merged" : "Deployed"} on {dayjs(version.deployedAt).format("MMMM D, YYYY @ h:mma")}. <a className="replicated-link" href={githubLink} rel="noopener noreferrer" target="_blank">View the PR</a>.</span>;
  } else if (gitRef) {
    deployedAtTextNode = <span className="gh-version-detail-text">Merged on date not available. <a className="replicated-link" href={githubLink} rel="noopener noreferrer" target="_blank">View the PR</a> to see when it was merged.</span>
  } else {
    deployedAtTextNode = "Deployed on date not available.";
  }
  let openedOnTextNode;
  if (version.createdOn) {
    openedOnTextNode = <span className="gh-version-detail-text u-fontSize--small u-fontWeight--medium u-color--dustyGray u-marginLeft--10 flex alignItems--center">Opened on {dayjs(version.createdOn).format("MMMM D, YYYY @ h:mma")}</span>;
  }
  return (
    <div className="flex u-paddingTop--20 u-paddingBottom--20 u-borderBottom--gray">
      <div className="flex-column flex1 u-paddingLeft--10">
        <div className="flex alignItems--center u-fontSize--larger u-color--tuna u-fontWeight--bold">
          Version {version.title}
          {prPending && openedOnTextNode}
          {isDeploying
            ? <Loader size="30" className="u-marginLeft--20" />
            : shipInstallnode
          }
        </div>
        {version.status === "deployed" || version.status === "merged" &&
          <p className="u-fontSize--small u-fontWeight--medium u-color--dustyGray u-marginTop--10 flex alignItems--center">
            {version.pullrequestNumber &&
              <span className="icon integration-card-icon-github u-marginRight--5" />
            }
            {deployedAtTextNode}
          </p>
        }
        {prPending &&
          <p className="u-fontSize--small u-fontWeight--medium u-color--dustyGray u-marginTop--10 flex alignItems--center">
            <span className="icon integration-card-icon-github u-marginRight--5" />
            <span className="gh-version-detail-text"><a className="replicated-link" href={githubLink} rel="noopener noreferrer" target="_blank">View this PR on GitHub</a> to review and merged it in for deployment.</span>
          </p>
        }
        <div className="u-marginTop--10">
          <Link to={isKots ? `/app/${urlParams.slug}/tree/${version.sequence}` : `/watch/${downstreamWatch.slug}/tree/${version.sequence}`} className="u-fontSize--small replicated-link">View file contents</Link>
          {version.releaseNotes && (
            <span className="u-paddingLeft--15"> |
              <span className="release-notes-link u-fontSize--small u-fontWeight--medium u-paddingLeft--15" onClick={() => { onReleaseNotesClick(version.releaseNotes); }}>Release notes</span>
            </span>
          )}
        </div>
      </div>
      <div className="flex flex-auto justifyContent--flexEnd alignItems--center u-paddingRight--10">
        <div className="">
          <div className="flex justifyContent--center alignItems--center">
            {version.status === "failed" && (
              <button 
                className="btn secondary u-marginRight--20" 
                onClick={() => {
                  if (props.handleViewLogs) {
                    props.handleViewLogs(version);
                  }
                }}
              >
                View Logs
              </button>
            )}
            {hasPreflight && version.status === "pending" && (
              <Link to={`/app/${urlParams.slug}/downstreams/${urlParams.downstreamSlug}/version-history/preflight/${version.sequence}`} className="u-fontSize--normal u-color--dustyGray">
                <button className="btn primary u-marginRight--20">
                  View Preflight Results
                </button>
              </Link>
            )}
            <div
              data-tip={`${version.title}-${version.sequence}`}
              data-for={`${version.title}-${version.sequence}`}
              className={classNames("icon", {
                "checkmark-icon": version.status === "deployed" || version.status === "merged",
                "exclamationMark--icon": version.status === "opened" || version.status === "pending",
                "grayCircleMinus--icon": version.status === "closed",
                "error-small": version.status === "failed"
              })}
            />
              <span className={classNames("u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginLeft--5", {
                "u-color--nevada": version.status === "deployed" || version.status === "merged",
                "u-color--orange": version.status === "opened" || version.status === "pending",
                "u-color--dustyGray": version.status === "closed",
                "u-color--red": version.status === "failed"
              })}>
                {Utilities.toTitleCase(version.status).replace("_", " ")}
              </span>
              {version.status === "pending_preflight" && (
                <span className="u-paddingLeft--5">
                  <Loader size="20" />
                </span>
              )}
          </div>
        </div>
      </div>
    </div>
  );
}
