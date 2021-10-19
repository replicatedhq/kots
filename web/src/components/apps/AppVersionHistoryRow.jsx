
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

function renderVersionAction(version, latestVersion, nothingToCommitDiff, app, history, deployVersion, showDownstreamReleaseNotes, viewLogs) {
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
  const isLatestVersion = version.sequence === latestVersion.sequence;
  const isPastVersion = find(downstream.pastVersions, { sequence: version.sequence });
  const isPendingDeployedVersion = find(downstream.pendingVersions, { sequence: version.sequence, status: "deployed" });
  const needsConfiguration = version.status === "pending_config";
  const showActions = !isPastVersion || app.allowRollback;
  const isRedeploy = isCurrentVersion && (version.status === "failed" || version.status === "deployed");
  const isRollback = isPastVersion && version.deployedAt && app.allowRollback;

  const isSecondaryBtn = isPastVersion || needsConfiguration || isRedeploy && !isRollback;
  const isPrimaryButton = !isSecondaryBtn && !isRedeploy && !isRollback;
  const editableConfig = isCurrentVersion || isLatestVersion;
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
      {version?.releaseNotes &&
        <div>
          <span className="icon releaseNotes--icon u-marginRight--10 u-cursor--pointer" onClick={() => showDownstreamReleaseNotes(version?.releaseNotes)} data-tip="View release notes" />
          <ReactTooltip effect="solid" className="replicated-tooltip" />
        </div>
      }
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
            {version.status === "failed" ? <span className="icon version-row-preflight-status-icon preflight-checks-failed-icon" /> : null}
          </div>
        : null}
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

function getUpdateTypeClassname(updateType) {
  if (updateType.includes("Upstream Update")) {
    return "upstream-update";
  }
  if (updateType.includes("Config Change")) {
    return "config-update";
  }
  if (updateType.includes("License Change")) {
    return "license-sync";
  }
  if (updateType.includes("Airgap Install") || updateType.includes("Airgap Update")) {
    return "airgap-install";
  }
  return "online-install";
}

function renderVersionStatus(version, app, match, viewLogs) {
  const downstream = app.downstreams?.length && app.downstreams[0];
  if (!downstream) {
    return null;
  }

  const isPastVersion = find(downstream.pastVersions, { sequence: version.sequence });
  const isPendingDeployedVersion = find(downstream.pendingVersions, { sequence: version.sequence, status: "deployed" });
  const clusterSlug = downstream.cluster?.slug;
  
  let preflightBlock = null;
  if (version.status === "pending_preflight") {
    preflightBlock = (
      <span className="flex u-marginLeft--5 alignItems--center">
        <Loader size="20" />
      </span>);
  } else if (app.hasPreflight) {
    preflightBlock = (<Link to={`/app/${match.params.slug}/downstreams/${clusterSlug}/version-history/preflight/${version.sequence}`} className="replicated-link u-marginLeft--5 u-fontSize--small">View preflights</Link>);
  }
  
  if (!isPastVersion && !isPendingDeployedVersion) {
    if (version.status === "deployed" || version.status === "merged") {
      return (
        <div>
          <span className="status-tag success flex-auto u-cursor--default" data-tip={version.deployedAt ? `Deployed ${Utilities.dateFormat(version.deployedAt, "MMMM D, YYYY @ hh:mm a z")}` : "Unable to find deployed at date"}>Currently {version.status.replace("_", " ")} version</span>
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
    } else {
      return <span className="status-tag unknown flex-atuo"> {Utilities.toTitleCase(version.status).replace("_", " ")} </span>
    }
  }
}

export default function AppVersionHistoryRow(props) {
  const { version, selectedDiffReleases, nothingToCommit,
    isChecked, isNew, renderSourceAndDiff, handleSelectReleasesToDiff,
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
          <p className="u-fontSize--header2 u-fontWeight--bold u-lineHeight--medium u-textColor--primary">{version.versionLabel || version.title}</p>
          <p className="u-fontSize--small u-textColor--bodyCopy u-fontWeight--medium u-marginLeft--10" style={{ marginTop: "2px" }}>Sequence {version.sequence}</p>
        </div>
        <p className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy u-marginTop--5"> Released <span className="u-fontWeight--bold">{version.upstreamReleasedAt ? Utilities.dateFormat(version.upstreamReleasedAt, "MM/DD/YY @ hh:mm a z") : Utilities.dateFormat(version.createdOn, "MM/DD/YY @ hh:mm a z")}</span></p>
        <div className="u-marginTop--5 flex flex-auto alignItems--center">
          <span className={`icon versionUpdateType u-marginRight--5 ${getUpdateTypeClassname(version.source)}`} data-tip={version.source} />
          <ReactTooltip effect="solid" className="replicated-tooltip" />
          {renderSourceAndDiff(version)}
          {yamlErrorsDetails && renderYamlErrors(yamlErrorsDetails, version, toggleShowDetailsModal)}
        </div>
      </div>
      <div className={`${nothingToCommit && selectedDiffReleases && "u-opacity--half"} flex-column flex1`}>
        <div className="flex flex1 alignItems--center"> {gitopsEnabled ? renderViewPreflights(version, props.app, props.match) : renderVersionStatus(version, props.app, props.match, props.handleViewLogs)}</div>
      </div>
      <div className={`${nothingToCommit && selectedDiffReleases && "u-opacity--half"} flex-column flex-auto alignItems--flexEnd justifyContent--center`}>
        <div>
          {version.status === "failed" || version.status === "deployed" ?
            renderVersionAction(version, latestVersion, nothingToCommit && selectedDiffReleases, props.app, props.history, props.redeployVersion, props.showDownstreamReleaseNotes, props.handleViewLogs) :
            renderVersionAction(version, latestVersion, nothingToCommit && selectedDiffReleases, props.app, props.history, props.deployVersion, props.showDownstreamReleaseNotes, props.handleViewLogs)
          }
        </div>
      </div>
    </div>
  )
}
