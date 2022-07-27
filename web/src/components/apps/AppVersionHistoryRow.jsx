import React, { Component } from "react";
import { Link } from "react-router-dom";
import find from "lodash/find";
import classNames from "classnames";
import ReactTooltip from "react-tooltip";

import Loader from "../shared/Loader";

import { Utilities, getPreflightResultState } from "../../utilities/utilities";

const YamlErrors = ({ version, handleSeeDetailsClicked }) => {
  if (!version.yamlErrors) {
    return null;
  }
  return (
    <div className="flex alignItems--center u-marginTop--5">
      <span className="icon error-small" />
      <span className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginLeft--5 u-textColor--error">
        {version.yamlErrors?.length} Invalid file
        {version.yamlErrors?.length !== 1 ? "s" : ""}{" "}
      </span>
      <span
        className="replicated-link u-marginLeft--5 u-fontSize--small"
        onClick={handleSeeDetailsClicked}
      >
        {" "}
        See details{" "}
      </span>
    </div>
  );
};

const AppVersionHistoryRow = ({
  handleActionButtonClicked,
  isHelmManaged,
  app,
  match,
  history,
  version,
  selectedDiffReleases,
  nothingToCommit,
  isChecked,
  isNew,
  newPreflightResults,
  showReleaseNotes,
  renderDiff,
  toggleShowDetailsModal,
  gitopsEnabled,
  deployVersion,
  redeployVersion,
  downloadVersion,
  upgradeAdminConsole,
  handleViewLogs,
  handleSelectReleasesToDiff,
  renderVersionDownloadStatus,
  isDownloading,
  adminConsoleMetadata,
  makeCurrentVersion,
  makingCurrentVersionErrMsg,
  updateCallback,
  toggleIsBundleUploading,
  isBundleUploading,
  refreshAppData,
  displayErrorModal,
  toggleErrorModal,
  makingCurrentRelease,
  redeployVersionErrMsg,
}) => {

  renderDiff = (_version) => {
    const hideSourceDiff =
      _version.source?.includes("Airgap Install") ||
      _version.source?.includes("Online Install");
    if (hideSourceDiff) {
      return null;
    }
    return (
      <div className="u-marginTop--5">{renderDiff(_version)}</div>
    );
  };

  handleSelectReleasesToDiff = () => {
    if (!selectedDiffReleases) {
      return;
    }
    if (nothingToCommit) {
      return;
    }
    handleSelectReleasesToDiff(
      version,
      !isChecked
    );
  };

  function deployButtonStatus(version) {
    const app = app;
    const downstream = app?.downstream;

    const isCurrentVersion =
      version.sequence === downstream.currentVersion?.sequence;
    const isDeploying = version.status === "deploying";
    const isPastVersion = find(downstream.pastVersions, {
      sequence: version.sequence,
    });
    const needsConfiguration = version.status === "pending_config";
    const isRollback = isPastVersion && version.deployedAt && app.allowRollback;
    const isRedeploy =
      isCurrentVersion &&
      (version.status === "failed" || version.status === "deployed");
    const canUpdateKots =
      version.needsKotsUpgrade &&
      !adminConsoleMetadata?.isAirgap &&
      !adminConsoleMetadata?.isKurl;

    if (needsConfiguration) {
      return "Configure";
    } else if (downstream?.currentVersion?.sequence == undefined) {
      if (canUpdateKots) {
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
      if (canUpdateKots) {
        return "Upgrade";
      } else {
        return "Deploy";
      }
    }
  }

  const getPreflightState = (version) => {
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
      preflightSkipped: version?.preflightSkipped,
    };
  };

  const renderReleaseNotes = (version) => {
    if (!version?.releaseNotes) {
      return null;
    }
    return (
      <div>
        <span
          className="icon releaseNotes--icon u-marginRight--10 u-cursor--pointer"
          onClick={() => showReleaseNotes(version?.releaseNotes)}
          data-tip="View release notes"
        />
        <ReactTooltip effect="solid" className="replicated-tooltip" />
      </div>
    );
  };

  const renderVersionAction = (version) => {
    const app = app;
    const downstream = app?.downstream;

    let actionFn = deployVersion;
    if (isHelmManaged) {
      actionFn = () => { };
    } else if (version.needsKotsUpgrade) {
      actionFn = upgradeAdminConsole;
    } else if (version.status === "pending_download") {
      actionFn = downloadVersion;
    } else if (version.status === "failed" || version.status === "deployed") {
      actionFn = redeployVersion;
    }

    if (version.status === "pending_download") {
      let buttonText = "Download";
      if (isDownloading) {
        buttonText = "Downloading";
      } else if (version.needsKotsUpgrade) {
        buttonText = "Upgrade";
      }
      return (
        <div className="flex flex1 justifyContent--flexEnd alignItems--center">
          {renderReleaseNotes(version)}
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

    const isCurrentVersion =
      version.sequence === downstream.currentVersion?.sequence;
    const isLatestVersion = version.sequence === app.currentSequence;
    const isPendingVersion = find(downstream.pendingVersions, {
      sequence: version.sequence,
    });
    const isPastVersion = find(downstream.pastVersions, {
      sequence: version.sequence,
    });
    const isPendingDeployedVersion = find(downstream.pendingVersions, {
      sequence: version.sequence,
      status: "deployed",
    });
    const needsConfiguration = version.status === "pending_config";
    const showActions = !isPastVersion || app.allowRollback;
    const isRedeploy =
      isCurrentVersion &&
      (version.status === "failed" || version.status === "deployed");
    const isRollback = isPastVersion && version.deployedAt && app.allowRollback;

    const isSecondaryBtn =
      isPastVersion || needsConfiguration || (isRedeploy && !isRollback);
    const isPrimaryButton = !isSecondaryBtn && !isRedeploy && !isRollback;
    const editableConfig =
      isCurrentVersion || isLatestVersion || isPendingVersion?.semver;

    let tooltipTip;
    if (editableConfig) {
      tooltipTip = "Edit config";
    } else {
      tooltipTip = "View config";
    }

    const preflightState = getPreflightState(version);
    let checksStatusText;
    if (preflightState.preflightsFailed) {
      checksStatusText = "Checks failed";
    } else if (preflightState.preflightState === "warn") {
      checksStatusText = "Checks passed with warnings";
    } else {
      checksStatusText = "Checks passed";
    }

    if (downstream.gitops?.enabled) {
      if (version.gitDeployable === false) {
        return (
          <div
            className={
              nothingToCommit &&
              selectedDiffReleases &&
              "u-opacity--half"
            }
          >
            Nothing to commit
          </div>
        );
      }
      if (!version.commitUrl) {
        return (
          <div className="flex flex1 justifyContent--flexEnd alignItems--center">
            {renderReleaseNotes(version)}
            <>
              {version.status === "pending_preflight" ? (
                <div className="u-position--relative">
                  <Loader size="30" />
                  <p className="checks-running-text u-fontSize--small u-lineHeight--normal u-fontWeight--medium">
                    Running checks
                  </p>
                </div>
              ) : preflightState.preflightState !== "" ? (
                <>
                  <Link
                    to={`/app/${app?.slug}/downstreams/${app?.downstream.cluster?.slug}/version-history/preflight/${version?.sequence}`}
                    className="icon preflightChecks--icon u-cursor--pointer u-position--relative"
                    data-tip="View preflight checks"
                  >
                    {preflightState.preflightsFailed ||
                      preflightState.preflightState === "warn" ||
                      newPreflightResults ? (
                      <div>
                        <span
                          className={`icon version-row-preflight-status-icon ${preflightState.preflightsFailed
                            ? "preflight-checks-failed-icon"
                            : preflightState.preflightState === "warn"
                              ? "preflight-checks-warn-icon"
                              : ""
                            }`}
                        />
                        <p
                          className={`checks-running-text u-fontSize--small u-lineHeight--normal u-fontWeight--medium ${preflightState.preflightsFailed
                            ? "err"
                            : preflightState.preflightState === "warn"
                              ? "warning"
                              : newPreflightResults
                                ? "success"
                                : ""
                            }`}
                        >
                          {checksStatusText}
                        </p>
                      </div>
                    ) : null}
                  </Link>
                  <ReactTooltip effect="solid" className="replicated-tooltip" />
                </>
              ) : null}
            </>
          </div>
        );
      }
      return (
        <div className="flex flex1 justifyContent--flexEnd alignItems--center">
          {renderReleaseNotes(version)}
          <div>
            {version.status === "pending_preflight" ? (
              <div className="u-position--relative">
                <Loader size="30" />
                <p className="checks-running-text u-fontSize--small u-lineHeight--normal u-fontWeight--medium">
                  Running checks
                </p>
              </div>
            ) : preflightState.preflightState !== "" ? (
              <>
                <Link
                  to={`/app/${app?.slug}/downstreams/${app?.downstream.cluster?.slug}/version-history/preflight/${version?.sequence}`}
                  className="icon preflightChecks--icon u-cursor--pointer u-position--relative"
                  data-tip="View preflight checks"
                >
                  {preflightState.preflightsFailed ||
                    preflightState.preflightState === "warn" ||
                    newPreflightResults ? (
                    <div>
                      <span
                        className={`icon version-row-preflight-status-icon ${preflightState.preflightsFailed
                          ? "preflight-checks-failed-icon"
                          : preflightState.preflightState === "warn"
                            ? "preflight-checks-warn-icon"
                            : ""
                          }`}
                      />
                      <p
                        className={`checks-running-text u-fontSize--small u-lineHeight--normal u-fontWeight--medium ${preflightState.preflightsFailed
                          ? "err"
                          : preflightState.preflightState === "warn"
                            ? "warning"
                            : newPreflightResults
                              ? "success"
                              : ""
                          }`}
                      >
                        {checksStatusText}
                      </p>
                    </div>
                  ) : null}
                </Link>
                <ReactTooltip effect="solid" className="replicated-tooltip" />
              </>
            ) : null}
          </div>
          <button
            className="btn primary blue u-marginLeft--10"
            onClick={() => window.open(version.commitUrl, "_blank")}
          >
            View commit
          </button>
        </div>
      );
    }

    return (
      <div className="flex flex1 justifyContent--flexEnd alignItems--center">
        {renderReleaseNotes(version)}

        <div>
          {version.status === "pending_preflight" ? (
            <div className="u-marginRight--10 u-position--relative">
              <Loader size="30" />
              <p className="checks-running-text u-fontSize--small u-lineHeight--normal u-fontWeight--medium">
                Running checks
              </p>
            </div>
          ) : preflightState.preflightState !== "" ? (
            <>
              <Link
                to={`/app/${app?.slug}/downstreams/${app?.downstream.cluster?.slug}/version-history/preflight/${version?.sequence}`}
                className="icon preflightChecks--icon u-marginRight--10 u-cursor--pointer u-position--relative"
                data-tip="View preflight checks"
              >
                {preflightState.preflightsFailed ||
                  preflightState.preflightState === "warn" ||
                  newPreflightResults ? (
                  <div>
                    <span
                      className={`icon version-row-preflight-status-icon ${preflightState.preflightsFailed
                        ? "preflight-checks-failed-icon"
                        : preflightState.preflightState === "warn"
                          ? "preflight-checks-warn-icon"
                          : ""
                        }`}
                    />
                    <p
                      className={`checks-running-text u-fontSize--small u-lineHeight--normal u-fontWeight--medium ${preflightState.preflightsFailed
                        ? "err"
                        : preflightState.preflightState === "warn"
                          ? "warning"
                          : newPreflightResults
                            ? "success"
                            : ""
                        }`}
                    >
                      {checksStatusText}
                    </p>
                  </div>
                ) : null}
              </Link>
              <ReactTooltip effect="solid" className="replicated-tooltip" />
            </>
          ) : null}
        </div>
        {app.isConfigurable && (
          <div className="flex alignItems--center">
            <Link
              to={`/app/${app.slug}/config/${version.sequence}`}
              className={`icon ${editableConfig ? "configEdit--icon" : "configView--icon"
                } u-cursor--pointer`}
              data-tip={tooltipTip}
            />
            <ReactTooltip effect="solid" className="replicated-tooltip" />
          </div>
        )}
        {(isPastVersion || isCurrentVersion || isPendingDeployedVersion) &&
          version?.status !== "pending" ? (
          <div className="u-marginLeft--10">
            <span
              className="icon deployLogs--icon u-cursor--pointer"
              onClick={() =>
                handleViewLogs(version, version?.status === "failed")
              }
              data-tip="View deploy logs"
            />
            <ReactTooltip effect="solid" className="replicated-tooltip" />
            {version.status === "failed" ? (
              <span className="icon version-row-preflight-status-icon preflight-checks-failed-icon logs" />
            ) : null}
          </div>
        ) : null}
        {showActions && (
          <div className="flex alignItems--center">
            <button
              className={classNames("btn u-marginLeft--10", {
                "secondary dark": isRollback,
                "secondary blue": isSecondaryBtn,
                "primary blue": isPrimaryButton,
              })}
              disabled={isActionButtonDisabled(version)}
              onClick={() => {
                handleActionButtonClicked();
                if (needsConfiguration) {
                  history.push(
                    `/app/${app.slug}/config/${version.sequence}`
                  );
                  return null;
                }
                if (isRollback) {
                  actionFn(version, true);
                  return null;
                }

                actionFn(version);
                return null;
              }}
            >
              <span
                key={version.nonDeployableCause}
                data-tip-disable={!isActionButtonDisabled(version)}
                data-tip={version.nonDeployableCause}
                data-for="disable-deployment-tooltip"
              >
                {deployButtonStatus(version)}
              </span>
            </button>
            <ReactTooltip effect="solid" id="disable-deployment-tooltip" />
          </div>
        )}
      </div>
    );
  };

  const isActionButtonDisabled = (version) => {
    if (isHelmManaged) {
      return false;
    }
    if (version.status === "deploying") {
      return true;
    }
    if (version.status === "pending_config") {
      return false;
    }
    if (version.status === "pending_download") {
      return false;
    }
    return !version.isDeployable;
  };

  const renderVersionStatus = (version) => {
    const app = app;
    const downstream = app?.downstream;
    if (!downstream) {
      return null;
    }

    const isPastVersion = find(downstream.pastVersions, {
      sequence: version.sequence,
    });
    const isPendingDeployedVersion = find(downstream.pendingVersions, {
      sequence: version.sequence,
      status: "deployed",
    });

    if (!isPastVersion && !isPendingDeployedVersion) {
      if (version.status === "deployed" || version.status === "merged") {
        return (
          <div>
            <span
              className="status-tag success flex-auto u-cursor--default"
              data-tip={
                version.deployedAt
                  ? `${version.status === "deploying"
                    ? "Deploy started at"
                    : "Deployed"
                  } ${Utilities.dateFormat(
                    version.deployedAt,
                    "MMMM D, YYYY @ hh:mm a z"
                  )}`
                  : "Unable to find deployed at date"
              }
            >
              Currently {version.status.replace("_", " ")} version
            </span>
            <ReactTooltip effect="solid" className="replicated-tooltip" />
            {version.preflightSkipped && (
              <p
                style={{ maxWidth: "200px" }}
                className="u-textColor--bodyCopy u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginTop--5"
              >
                This version was deployed before preflight checks had completed
              </p>
            )}
          </div>
        );
      } else if (version.status === "failed") {
        return (
          <div className="flex alignItems--center">
            <span className="status-tag failed flex-auto u-marginRight--10">
              Deploy Failed
            </span>
            <span
              className="replicated-link u-fontSize--small"
              onClick={() => handleViewLogs(version, true)}
            >
              View deploy logs
            </span>
            {version.preflightSkipped && (
              <p
                style={{ maxWidth: "200px" }}
                className="u-textColor--bodyCopy u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginTop--5"
              >
                This version was deployed before preflight checks had completed
              </p>
            )}
          </div>
        );
      } else if (version.status === "deploying") {
        return (
          <span className="flex alignItems--center u-fontSize--small u-lineHeight--normal u-textColor--bodyCopy u-fontWeight--medium">
            <Loader
              className="flex alignItems--center u-marginRight--5"
              size="16"
            />
            Deploying
          </span>
        );
      } else if (version.status !== "pending") {
        return (
          <span className="status-tag unknown flex-atuo">
            {" "}
            {Utilities.toTitleCase(version.status).replace("_", " ")}{" "}
          </span>
        );
      }
    } else {
      if (version.status === "deployed" || version.status === "merged") {
        return (
          <div>
            <span
              className="status-tag unknown flex-auto u-cursor--default"
              data-tip={
                version.deployedAt
                  ? `Deployed ${Utilities.dateFormat(
                    version.deployedAt,
                    "MMMM D, YYYY @ hh:mm a z"
                  )}`
                  : "Unable to find deployed at date"
              }
            >
              Previously deployed
            </span>
            <ReactTooltip effect="solid" className="replicated-tooltip" />
          </div>
        );
      } else if (version.status === "pending") {
        return (
          <span className="status-tag skipped flex-auto">Version skipped</span>
        );
      } else if (version.status === "failed") {
        return (
          <div className="flex alignItems--center">
            <span className="status-tag failed flex-auto u-marginRight--10">
              Deploy Failed
            </span>
            <span
              className="replicated-link u-fontSize--small"
              onClick={() => handleViewLogs(version, true)}
            >
              View deploy logs
            </span>
          </div>
        );
      } else if (version.status === "deploying") {
        return (
          <span className="flex alignItems--center u-fontSize--small u-lineHeight--normal u-textColor--bodyCopy u-fontWeight--medium">
            <Loader
              className="flex alignItems--center u-marginRight--5"
              size="16"
            />
            Deploying
          </span>
        );
      } else if (version.status === "pending_download") {
        return (
          <div className="flex alignItems--center">
            <span className="status-tag unknown flex-auto u-marginRight--10">
              Pending download
            </span>
          </div>
        );
      } else {
        return (
          <span className="status-tag unknown flex-auto">
            {" "}
            {Utilities.toTitleCase(version.status).replace("_", " ")}{" "}
          </span>
        );
      }
    }
  };

  return (
    <div
      key={version.sequence}
      className={classNames(
        `VersionHistoryRowWrapper ${version.status} flex-column justifyContent--center`,
        {
          overlay: selectedDiffReleases,
          disabled: nothingToCommit,
          selected: isChecked && !nothingToCommit,
          "is-new": isNew,
          "show-preflight-passed-text": newPreflightResults,
        }
      )}
      onClick={handleSelectReleasesToDiff}
    >
      <div className="VersionHistoryRow flex flex-auto">
        {selectedDiffReleases && (
          <div
            className={classNames(
              "checkbox u-marginRight--20",
              { checked: isChecked && !nothingToCommit },
              { disabled: nothingToCommit }
            )}
          />
        )}
        <div
          className={`${nothingToCommit && selectedDiffReleases && "u-opacity--half"
            } flex-column flex1 u-paddingRight--20`}
        >
          <div className="flex alignItems--center">
            <p className="u-fontSize--header2 u-fontWeight--bold u-lineHeight--medium u-textColor--primary">
              {version.versionLabel || version.title}
            </p>
            <p
              className="u-fontSize--small u-textColor--bodyCopy u-fontWeight--medium u-marginLeft--10"
              style={{ marginTop: "2px" }}
            >
              Sequence {version.sequence}
            </p>
            {version.isRequired && (
              <span className="status-tag required u-marginLeft--10">
                {" "}
                Required{" "}
              </span>
            )}
          </div>
          <p className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy u-marginTop--5">
            {" "}
            Released{" "}
            <span className="u-fontWeight--bold">
              {version.upstreamReleasedAt
                ? Utilities.dateFormat(
                  version.upstreamReleasedAt,
                  "MM/DD/YY @ hh:mm a z"
                )
                : Utilities.dateFormat(
                  version.createdOn,
                  "MM/DD/YY @ hh:mm a z"
                )}
            </span>
          </p>
          {renderDiff(version)}
          <YamlErrors
          version={version}
          handleSeeDetailsClicked={() => toggleShowDetailsModal(version.yamlErrors, version.sequence)}
          />
        </div>
        <div
          className={`${nothingToCommit && selectedDiffReleases && "u-opacity--half"
            } flex-column flex1 justifyContent--center`}
        >
          <p className="u-fontSize--small u-fontWeight--bold u-textColor--lightAccent u-lineHeight--default">
            {version.source}
          </p>
          {gitopsEnabled && version.status !== "pending_download" ? null : (
            <div className="flex flex-auto u-marginTop--10">
              {renderVersionStatus(version)}
            </div>
          )}
        </div>
        <div
          className={`${nothingToCommit && selectedDiffReleases && "u-opacity--half"
            } flex-column flex-auto alignItems--flexEnd justifyContent--center`}
        >
          {renderVersionAction(version)}
        </div>
      </div>
      {renderVersionDownloadStatus(version)}
    </div>
  );
}

export { AppVersionHistoryRow };
