
import React from "react";
import { Link } from "react-router-dom";
import find from "lodash/find";
import classNames from "classnames";
import ReactTooltip from "react-tooltip";

import Loader from "../shared/Loader";

import { Utilities, getPreflightResultState } from "../../utilities/utilities";

function renderYamlErrors(yamlErrorsDetails, version, toggleShowDetailsModal) {
  return (
    <div className="flex alignItems--center u-marginLeft--5">
      <span className="icon error-small" />
      <span className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginLeft--5 u-color--red">{yamlErrorsDetails?.length} Invalid file{yamlErrorsDetails?.length !== 1 ? "s" : ""} </span>
      <span className="replicated-link u-marginLeft--5 u-fontSize--small" onClick={() => toggleShowDetailsModal(yamlErrorsDetails, version.sequence)}> See details </span>
    </div>
  )
}

function deployButtonStatus(downstream, version, app) {
  const isCurrentVersion = version.sequence === downstream.currentVersion?.sequence;
  const isDeploying = version.status === "deploying";
  const isPastVersion = find(downstream.pastVersions, { sequence: version.sequence });
  const needsConfiguration = version.status === "pending_config";
  const isRollback = isPastVersion && version.deployedAt && app.allowRollback;
  const isRedeploy = isCurrentVersion && (version.status === "failed" || version.status === "deployed");

  if (needsConfiguration) {
    return "Configure";
  } else if (downstream?.currentVersion?.sequence == undefined) {
    return "Deploy";
  } else if (isRedeploy) {
    return "Redeploy";
  } else if (isRollback) {
    return "Rollback";
  } else if (isDeploying) {
    return "Deploying";
  } else if (isCurrentVersion) {
    return "Deployed";
  } else {
    return "Deploy";
  }
}

function isVersionEditable(latestVersion, version) {
  if (latestVersion?.sequence <= version?.parentSequence) {
    return true;
  } else {
    return false;
  }
}

function renderVersionAction(version, latestVersion, nothingToCommitDiff, app, history, deployVersion) {
  const downstream = app.downstreams[0];

  if (downstream.gitops?.enabled) {
    if (version.gitDeployable === false) {
      return (<div className={nothingToCommitDiff && "u-opacity--half"}>Nothing to commit</div>);
    }
    if (!version.commitUrl) {
      return null;
    }
    return (
      <button
        className="btn primary blue"
        onClick={() => window.open(version.commitUrl, '_blank')}
      >
        View
      </button>
    );
  }

  const isCurrentVersion = version.sequence === downstream.currentVersion?.sequence;
  const isPastVersion = find(downstream.pastVersions, { sequence: version.sequence });
  const needsConfiguration = version.status === "pending_config";
  const showActions = !isPastVersion || app.allowRollback;
  const isRedeploy = isCurrentVersion && (version.status === "failed" || version.status === "deployed");
  const isRollback = isPastVersion && version.deployedAt && app.allowRollback;

  const isSecondaryBtn = isPastVersion || needsConfiguration || isRedeploy && !isRollback;
  const isPrimaryButton = !isSecondaryBtn && !isRedeploy && !isRollback;
  let tooltipTip;
  if (isVersionEditable(latestVersion, version)) {
    tooltipTip = "Edit config";
  } else {
    tooltipTip = "View config"
  }

  return (
    <div className="flex flex1 justifyContent--flexEnd">
      {app.isConfigurable &&
        <div className="flex alignItems--center">
          <Link to={`/app/${app.slug}/config/${version.sequence}`} className="icon config--icon u-cursor--pointer" data-tip={tooltipTip} />
          <ReactTooltip effect="solid" className="replicated-tooltip" />
        </div>}
      {showActions &&
        <button
          className={classNames("btn u-marginLeft--10", { "secondary dark": isRollback, "secondary blue": isSecondaryBtn, "primary blue": isPrimaryButton })}
          disabled={version.status === "deploying"}
          onClick={() => needsConfiguration ? history.push(`/app/${app.slug}/config/${version.sequence}`) : isRollback ? deployVersion(version, true) : deployVersion(version)}
        >
          {deployButtonStatus(downstream, version, app)}
        </button>
      }
    </div>
  );
}

function renderViewPreflights(version, app, match) {
  const downstream = app.downstreams[0];
  const clusterSlug = downstream.cluster?.slug;
  return (
    <Link className="u-marginTop--10" to={`/app/${match.params.slug}/downstreams/${clusterSlug}/version-history/preflight/${version?.sequence}`}>
      <span className="replicated-link" style={{ fontSize: 12 }}>View preflight results</span>
    </Link>
  );
}

function renderVersionStatus(version, app, match, viewLogs) {
  const downstream = app.downstreams?.length && app.downstreams[0];
  if (!downstream) {
    return null;
  }

  let preflightsFailed = false;
  if (version.status === "pending" && version.preflightResult) {
    const preflightResult = JSON.parse(version.preflightResult);
    const preflightState = getPreflightResultState(preflightResult);
    preflightsFailed = preflightState === "fail";
  }

  const isPastVersion = find(downstream.pastVersions, { sequence: version.sequence });
  const isPendingDeployedVersion = find(downstream.pendingVersions, { sequence: version.sequence, status: "deployed" });
  const clusterSlug = downstream.cluster?.slug;
  let preflightBlock = null;

  if (isPastVersion && app.hasPreflight) {
    if (preflightsFailed) {
      preflightBlock = (<Link to={`/app/${match.params.slug}/downstreams/${clusterSlug}/version-history/preflight/${version.sequence}`} className="replicated-link u-marginLeft--5 u-fontSize--small">See details</Link>);
    } else if (version.status !== "pending_config") {
      preflightBlock = (<Link to={`/app/${match.params.slug}/downstreams/${clusterSlug}/version-history/preflight/${version.sequence}`} className="replicated-link u-marginLeft--5 u-fontSize--small">View preflights</Link>);
    }
  }
  if (version.status === "pending_preflight") {
    preflightBlock = (
      <span className="flex u-marginLeft--5 alignItems--center">
        <Loader size="20" />
      </span>);
  } else if (app.hasPreflight) {
    if (preflightsFailed) {
      preflightBlock = (<Link to={`/app/${match.params.slug}/downstreams/${clusterSlug}/version-history/preflight/${version.sequence}`} className="replicated-link u-marginLeft--5 u-fontSize--small">See details</Link>);
    } else if (version.status !== "pending_config") {
      preflightBlock = (<Link to={`/app/${match.params.slug}/downstreams/${clusterSlug}/version-history/preflight/${version.sequence}`} className="replicated-link u-marginLeft--5 u-fontSize--small">View preflights</Link>);
    }
  }

  if (!isPastVersion && !isPendingDeployedVersion) {
    return (
      <div className="flex alignItems--center">
        <div className="flex alignItems--center">
          <div
            data-tip={`${version.versionLabel || version.title}-${version.sequence}`}
            data-for={`${version.versionLabel || version.title}-${version.sequence}`}
            className={classNames("icon", {
              "checkmark-icon": version.status === "deployed" || version.status === "merged" || version.status === "pending",
              "exclamationMark--icon": version.status === "opened",
              "grayCircleMinus--icon": version.status === "closed",
              "error-small": version.status === "failed" || preflightsFailed
            })}
          />
          {version.status === "deploying" && <Loader className="flex alignItems--center" size="20" />}
          <span className={classNames("u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginLeft--5", {
            "u-color--nevada": version.status === "deployed" || version.status === "merged",
            "u-color--orange": version.status === "opened",
            "u-color--dustyGray": version.status === "closed" || version.status === "pending" || version.status === "pending_preflight",
            "u-color--red": version.status === "failed" || preflightsFailed
          })}>
            {Utilities.toTitleCase(
              version.status === "pending_preflight"
                ? "Running checks"
                : preflightsFailed
                  ? "Checks failed"
                  : version.status === "pending"
                    ? "Ready to deploy"
                    : version.status
            ).replace("_", " ")}
          </span>
        </div>
        {preflightBlock}
        {version.status === "failed" &&
          <span className="replicated-link u-marginLeft--5 u-fontSize--small" onClick={() => viewLogs(version, true)}>View logs</span>
        }
      </div>
    );
  } else {
    return (
      <div className="flex alignItems--center">
        <div className="flex alignItems--center">
          <div
            data-tip={`${version.versionLabel || version.title}-${version.sequence}`}
            data-for={`${version.versionLabel || version.title}-${version.sequence}`}
            className={classNames("icon", {
              "analysis-gray_checkmark": version.status === "deployed" || version.status === "merged",
              "exclamationMark--icon": version.status === "opened",
              "grayCircleMinus--icon": version.status === "closed" || version.status === "pending",
              "error-small": version.status === "failed"
            })}
          />
          <span className={classNames("u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginLeft--5", {
            "u-color--nevada": version.status === "deployed" || version.status === "merged",
            "u-color--orange": version.status === "opened",
            "u-color--dustyGray": version.status === "closed" || version.status === "pending" || version.status === "pending_preflight",
            "u-color--red": version.status === "failed"
          })}>
            {version.status === "deployed" ?
              "Previously Deployed" :
              version.status === "pending" ?
                "Skipped" :
                version.status === "failed" ?
                  "Failed" : ""}
          </span>
        </div>
        {preflightBlock}
        {version.status === "failed" &&
          <span className="replicated-link u-marginLeft--5 u-fontSize--small" onClick={() => viewLogs(version, true)}>View logs</span>
        }
      </div>
    );
  }
}

export default function AppVersionHistoryRow(props) {
  const { version, selectedDiffReleases, nothingToCommit,
    isChecked, isNew, showDownstreamReleaseNotes, renderSourceAndDiff, handleSelectReleasesToDiff,
    yamlErrorsDetails, gitopsEnabled, toggleShowDetailsModal, latestVersion } = props;


  return (
    <div
      key={version.sequence}
      className={classNames(`VersionHistoryDeploymentRow ${version.status} flex flex-auto`, { "overlay": selectedDiffReleases, "disabled": nothingToCommit, "selected": (isChecked && !nothingToCommit), "is-new": isNew })}
      onClick={() => selectedDiffReleases && !nothingToCommit && handleSelectReleasesToDiff(version, !isChecked)}
    >
      {selectedDiffReleases && <div className={classNames("checkbox u-marginRight--20", { "checked": (isChecked && !nothingToCommit) }, { "disabled": nothingToCommit })} />}
      <div className={`${nothingToCommit && selectedDiffReleases && "u-opacity--half"} flex-column flex1 u-paddingRight--20`}>
        <div className="flex alignItems--center">
          <p className="u-fontSize--large u-fontWeight--bold u-lineHeight--medium u-color--tuna">{version.versionLabel || version.title}</p>
          <p className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-color--tundora u-marginLeft--5" style={{ marginTop: "2px" }}>Sequence {version.sequence}</p>
        </div>
        <div className="flex alignItems--center u-marginTop--10"></div>
        <div className="flex flex1 u-marginTop--15 alignItems--center">
          <p className="u-fontSize--small u-lineHeight--normal u-color--dustyGray u-fontWeight--medium">Released <span className="u-fontWeight--bold">{version.upstreamReleasedAt ? Utilities.dateFormat(version.upstreamReleasedAt, "MMMM D, YYYY") : Utilities.dateFormat(version.createdOn, "MMMM D, YYYY")}</span></p>
          {version.releaseNotes ?
            <p className="release-notes-link u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginLeft--10 flex alignItems--center" onClick={() => showDownstreamReleaseNotes(version.releaseNotes)}> <span className="icon releaseNotes-small--icon clickable u-marginRight--5" />Release notes</p> : null}
        </div>
      </div>
      <div className={`${nothingToCommit && selectedDiffReleases && "u-opacity--half"} flex-column flex1`}>
        <div className="flex flex-column">
          <p className="u-fontSize--normal u-fontWeight--bold u-color--tuna">{version.source}</p>
          <div className="flex alignItems--center u-fontSize--small u-marginTop--10 u-color--dustyGray">
            {renderSourceAndDiff(version)}
            {yamlErrorsDetails && renderYamlErrors(yamlErrorsDetails, version, toggleShowDetailsModal)}
          </div>
        </div>
        <div className="flex flex1 alignItems--flexEnd"> {gitopsEnabled ? renderViewPreflights(version, props.app, props.match) : renderVersionStatus(version, props.app, props.match, props.handleViewLogs)}</div>
      </div>
      <div className={`${nothingToCommit && selectedDiffReleases && "u-opacity--half"} flex-column flex1 alignItems--flexEnd`}>
        <div>
          {version.status === "failed" || version.status === "deployed" ?
            renderVersionAction(version, latestVersion, nothingToCommit && selectedDiffReleases, props.app, props.history, props.redeployVersion) :
            renderVersionAction(version, latestVersion, nothingToCommit && selectedDiffReleases, props.app, props.history, props.deployVersion)
          }
        </div>
        <p className="u-fontSize--small u-lineHeight--normal u-color--dustyGray u-fontWeight--medium u-marginTop--15">Deployed: <span className="u-fontWeight--bold">{version.deployedAt ? Utilities.dateFormat(version.deployedAt, "MMMM D, YYYY @ hh:mm a z") : "N/A"}</span></p>
      </div>
    </div>
  )
}
