
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
      <span className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginLeft--5 u-textColor--error">{yamlErrorsDetails?.length} Invalid file{yamlErrorsDetails?.length !== 1 ? "s" : ""} </span>
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
    if (version.needsKotsUpgrade) {
      return "Upgrade";
    } else {
      return "Deploy";
    }
  } else if (isRedeploy) {
    return "Redeploy";
  } else if (isRollback) {
    return "Rollback";
  } else if (isDeploying) {
    return "Deploying";
  } else if (isCurrentVersion) {
    return "Deployed";
  } else {
    if (version.needsKotsUpgrade) {
      return "Upgrade";
    } else {
      return "Deploy";
    }
  }
}

function getPreflightState(version) {
  let preflightsFailed = false;
  let preflightState = "";
  if (version?.preflightResult) {
    const preflightResult = JSON.parse(version.preflightResult);
    preflightState = getPreflightResultState(preflightResult);
    preflightsFailed = preflightState === "fail";
  }
  return {
    preflightsFailed,
    preflightState,
    preflightSkipped: version?.preflightSkipped
  };
}

function renderReleaseNotes(version, showReleaseNotes) {
  if (!version?.releaseNotes) {
    return null;
  }
  return (
    <div>
      <span className="icon releaseNotes--icon u-marginRight--10 u-cursor--pointer" onClick={() => showReleaseNotes(version?.releaseNotes)} data-tip="View release notes" />
      <ReactTooltip effect="solid" className="replicated-tooltip" />
    </div>
  );
}

function renderVersionAction(version, nothingToCommitDiff, app, history, actionFn, showReleaseNotes, viewLogs, isDownloading) {
  const downstream = app.downstreams[0];

  if (version.status === "pending_download") {

    let buttonText = "Download";
    if (isDownloading) {
      buttonText = "Downloading";
    } else if (version.needsKotsUpgrade) {
      buttonText = "Upgrade";
    }

    return (
      <div className="flex flex1 justifyContent--flexEnd alignItems--center">
        {renderReleaseNotes(version, showReleaseNotes)}
        <button
          className={"btn secondary blue"}
          disabled={isDownloading}
          onClick={() => actionFn(version)}
        >
          {buttonText}
        </button>
      </div>
    );
  }

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
        onClick={() => window.open(version.commitUrl, "_blank")}
      >
        View
      </button>
    );
  }

  const isCurrentVersion = version.sequence === downstream.currentVersion?.sequence;
  const isLatestVersion = version.sequence === app.currentSequence;
  const isPendingVersion = find(downstream.pendingVersions, { sequence: version.sequence });
  const isPastVersion = find(downstream.pastVersions, { sequence: version.sequence });
  const isPendingDeployedVersion = find(downstream.pendingVersions, { sequence: version.sequence, status: "deployed" });
  const needsConfiguration = version.status === "pending_config";
  const showActions = !isPastVersion || app.allowRollback;
  const isRedeploy = isCurrentVersion && (version.status === "failed" || version.status === "deployed");
  const isRollback = isPastVersion && version.deployedAt && app.allowRollback;

  const isSecondaryBtn = isPastVersion || needsConfiguration || isRedeploy && !isRollback;
  const isPrimaryButton = !isSecondaryBtn && !isRedeploy && !isRollback;
  const editableConfig = isCurrentVersion || isLatestVersion || isPendingVersion?.semver;
  let tooltipTip;
  if (editableConfig) {
    tooltipTip = "Edit config";
  } else {
    tooltipTip = "View config"
  }
  const preflightState = getPreflightState(version);
  let checksStatusText;
  if (preflightState.preflightsFailed) {
    checksStatusText = "Checks failed"
  } else if (preflightState.preflightState === "warn") {
    checksStatusText = "Checks passed with warnings"
  }
  return (
    <div className="flex flex1 justifyContent--flexEnd alignItems--center">
      {renderReleaseNotes(version, showReleaseNotes)}

      <div>
        {version.status === "pending_preflight" ?
          <div className="u-marginRight--10 u-position--relative">
            <Loader size="30" />
            <p className="checks-running-text u-fontSize--small u-lineHeight--normal u-fontWeight--medium">Running checks</p>
          </div>
        :
        <div>
          <Link to={`/app/${app?.slug}/downstreams/${app?.downstreams[0].cluster?.slug}/version-history/preflight/${version?.sequence}`}
            className="icon preflightChecks--icon u-marginRight--10 u-cursor--pointer u-position--relative"
            data-tip="View preflight checks">
            {preflightState.preflightsFailed || preflightState.preflightState === "warn" ?
              <div>
                <span className={`icon version-row-preflight-status-icon ${preflightState.preflightsFailed ? "preflight-checks-failed-icon" : preflightState.preflightState === "warn" ? "preflight-checks-warn-icon" : ""}`} />
                <p className={`checks-running-text u-fontSize--small u-lineHeight--normal u-fontWeight--medium ${preflightState.preflightsFailed ? "err" : preflightState.preflightState === "warn" ? "warning" : ""}`}>{checksStatusText}</p>
              </div>
              : null}
          </Link>
          <ReactTooltip effect="solid" className="replicated-tooltip" />
        </div>
        }
      </div>
      {app.isConfigurable &&
        <div className="flex alignItems--center">
          <Link to={`/app/${app.slug}/config/${version.sequence}`} className={`icon ${editableConfig ? "configEdit--icon" : "configView--icon"} u-cursor--pointer`} data-tip={tooltipTip} />
          <ReactTooltip effect="solid" className="replicated-tooltip" />
        </div>}
        {(isPastVersion || isCurrentVersion || isPendingDeployedVersion) && version?.status !== "pending" ?
          <div className="u-marginLeft--10">
            <span className="icon deployLogs--icon u-cursor--pointer" onClick={() => viewLogs(version, version?.status === "failed")} data-tip="View deploy logs" />
            <ReactTooltip effect="solid" className="replicated-tooltip" />
            {version.status === "failed" ? <span className="icon version-row-preflight-status-icon preflight-checks-failed-icon logs" /> : null}
          </div>
        : null}
      {showActions &&
        <button
          className={classNames("btn u-marginLeft--10", { "secondary dark": isRollback, "secondary blue": isSecondaryBtn, "primary blue": isPrimaryButton })}
          disabled={version.status === "deploying"}
          onClick={() => needsConfiguration ? history.push(`/app/${app.slug}/config/${version.sequence}`) : isRollback ? actionFn(version, true) : actionFn(version)}
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

function renderVersionStatus(version, app, viewLogs) {
  const downstream = app.downstreams?.length && app.downstreams[0];
  if (!downstream) {
    return null;
  }

  const isPastVersion = find(downstream.pastVersions, { sequence: version.sequence });
  const isPendingDeployedVersion = find(downstream.pendingVersions, { sequence: version.sequence, status: "deployed" });
  
  if (!isPastVersion && !isPendingDeployedVersion) {
    if (version.status === "deployed" || version.status === "merged") {
      return (
        <div>
          <span className="status-tag success flex-auto u-cursor--default" data-tip={version.deployedAt ? `${version.status === "deploying" ? "Deploy started at" : "Deployed"} ${Utilities.dateFormat(version.deployedAt, "MMMM D, YYYY @ hh:mm a z")}` : "Unable to find deployed at date"}>Currently {version.status.replace("_", " ")} version</span>
          <ReactTooltip effect="solid" className="replicated-tooltip" />
          {version.preflightSkipped && <p style={{ maxWidth: "200px" }} className="u-textColor--bodyCopy u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginTop--5">This version was deployed before preflight checks had completed</p>}
        </div>
      )
    } else if (version.status === "failed") {
      return (
        <div className="flex alignItems--center">
          <span className="status-tag failed flex-auto u-marginRight--10">Deploy Failed</span>
          <span className="replicated-link u-fontSize--small" onClick={() => viewLogs(version, true)}>View deploy logs</span>
          {version.preflightSkipped && <p style={{ maxWidth: "200px" }} className="u-textColor--bodyCopy u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginTop--5">This version was deployed before preflight checks had completed</p>}
        </div>
      );
    } else if (version.status === "deploying") {
      return (
        <span className="flex alignItems--center u-fontSize--small u-lineHeight--normal u-textColor--bodyCopy u-fontWeight--medium">
          <Loader className="flex alignItems--center u-marginRight--5" size="16" />
            Deploying
        </span>);
    } else if (version.status !== "pending") {
      return <span className="status-tag unknown flex-atuo"> {Utilities.toTitleCase(version.status).replace("_", " ")} </span>
    }
  } else {
    if (version.status === "deployed" || version.status === "merged") {
      return (
        <div>
          <span className="status-tag unknown flex-auto u-cursor--default" data-tip={version.deployedAt ? `Deployed ${Utilities.dateFormat(version.deployedAt, "MMMM D, YYYY @ hh:mm a z")}` : "Unable to find deployed at date"}>Previously deployed</span>
          <ReactTooltip effect="solid" className="replicated-tooltip" />
        </div>
      )
    } else if (version.status === "pending") {
      return <span className="status-tag skipped flex-auto">Version skipped</span>
    }
    else if (version.status === "failed") {
      return (
        <div className="flex alignItems--center">
          <span className="status-tag failed flex-auto u-marginRight--10">Deploy Failed</span>
          <span className="replicated-link u-fontSize--small" onClick={() => viewLogs(version, true)}>View deploy logs</span>
        </div>
      );
    } else if (version.status === "deploying") {
      return (
        <span className="flex alignItems--center u-fontSize--small u-lineHeight--normal u-textColor--bodyCopy u-fontWeight--medium">
          <Loader className="flex alignItems--center u-marginRight--5" size="16" />
            Deploying
        </span>);
    } else if (version.status === "pending_download") {
      return (
        <div className="flex alignItems--center">
          <span className="status-tag unknown flex-auto u-marginRight--10">Pending download</span>
        </div>
      );
    } else {
      return <span className="status-tag unknown flex-auto"> {Utilities.toTitleCase(version.status).replace("_", " ")} </span>
    }
  }
}

export default function AppVersionHistoryRow(props) {
  const { version, selectedDiffReleases, nothingToCommit,
    isChecked, isNew, renderSourceAndDiff, handleSelectReleasesToDiff,
    yamlErrorsDetails, gitopsEnabled, toggleShowDetailsModal,
    renderVersionDownloadStatus, isDownloading } = props;
  
  const hideSourceDiff = version?.source.includes("Airgap Install") || version?.source.includes("Online Install");

  let actionFn = props.deployVersion;
  if (version.status === "failed" || version.status === "deployed") {
    actionFn = props.redeployVersion;
  } else if (version.needsKotsUpgrade) {
    actionFn = props.upgradeAdminConsole;
  } else if (version.status === "pending_download") {
    actionFn = props.downloadVersion;
  }

  return (
    <div
      key={version.sequence}
      className={classNames(`VersionHistoryRowWrapper ${version.status} flex-column flex-auto`, { "overlay": selectedDiffReleases, "disabled": nothingToCommit, "selected": (isChecked && !nothingToCommit), "is-new": isNew })}
      onClick={() => selectedDiffReleases && !nothingToCommit && handleSelectReleasesToDiff(version, !isChecked)}
    >
      <div className="VersionHistoryRow flex flex-auto">
        {selectedDiffReleases && <div className={classNames("checkbox u-marginRight--20", { "checked": (isChecked && !nothingToCommit) }, { "disabled": nothingToCommit })} />}
        <div className={`${nothingToCommit && selectedDiffReleases && "u-opacity--half"} flex-column flex1 u-paddingRight--20`}>
          <div className="flex alignItems--center">
            <p className="u-fontSize--header2 u-fontWeight--bold u-lineHeight--medium u-textColor--primary">{version.versionLabel || version.title}</p>
            <p className="u-fontSize--small u-textColor--bodyCopy u-fontWeight--medium u-marginLeft--10" style={{ marginTop: "2px" }}>Sequence {version.sequence}</p>
          </div>
          <p className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy u-marginTop--5"> Released <span className="u-fontWeight--bold">{version.upstreamReleasedAt ? Utilities.dateFormat(version.upstreamReleasedAt, "MM/DD/YY @ hh:mm a z") : Utilities.dateFormat(version.createdOn, "MM/DD/YY @ hh:mm a z")}</span></p>
          <div className="u-marginTop--5 flex flex-auto alignItems--center">
            {hideSourceDiff ? null : renderSourceAndDiff(version)}
            {yamlErrorsDetails && renderYamlErrors(yamlErrorsDetails, version, toggleShowDetailsModal)}
          </div>
        </div>
        <div className={`${nothingToCommit && selectedDiffReleases && "u-opacity--half"} flex-column flex1 justifyContent--center`}>
          <p className="u-fontSize--small u-fontWeight--bold u-textColor--lightAccent u-lineHeight--default">{version.source}</p>
          <div className="flex flex-auto u-marginTop--10"> {gitopsEnabled && version.status !== "pending_download" ? renderViewPreflights(version, props.app, props.match) : renderVersionStatus(version, props.app, props.handleViewLogs)}</div>
        </div>
        <div className={`${nothingToCommit && selectedDiffReleases && "u-opacity--half"} flex-column flex-auto alignItems--flexEnd justifyContent--center`}>
          {renderVersionAction(version, nothingToCommit && selectedDiffReleases, props.app, props.history, actionFn, props.showReleaseNotes, props.handleViewLogs, isDownloading)}
        </div>
      </div>
      {renderVersionDownloadStatus(version)}
    </div>
  )
}
