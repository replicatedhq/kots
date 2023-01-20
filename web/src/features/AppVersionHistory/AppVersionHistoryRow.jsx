import React, { Component } from "react";
import { Link } from "react-router-dom";
import find from "lodash/find";
import classNames from "classnames";
import ReactTooltip from "react-tooltip";

import Loader from "../../components/shared/Loader";

import { Utilities, getPreflightResultState } from "@src/utilities/utilities";

import { YamlErrors } from "./YamlErrors";
import Icon from "@src/components/Icon";
import EditConfigIcon from "@components/shared/EditConfigIcon";

class AppVersionHistoryRow extends Component {
  renderDiff = (version) => {
    const hideSourceDiff =
      version.source?.includes("Airgap Install") ||
      version.source?.includes("Online Install");
    if (hideSourceDiff) {
      return null;
    }
    return (
      <div className="u-marginTop--5">{this.props.renderDiff(version)}</div>
    );
  };

  handleSelectReleasesToDiff = () => {
    if (!this.props.selectedDiffReleases) {
      return;
    }
    if (this.props.nothingToCommit) {
      return;
    }
    this.props.handleSelectReleasesToDiff(
      this.props.version,
      !this.props.isChecked
    );
  };

  deployButtonStatus = (version) => {
    if (this.props.isHelmManaged) {
      const deployedSequence =
        this.props.app?.downstream?.currentVersion?.sequence;

      if (version.sequence > deployedSequence) {
        return "Deploy";
      }

      if (version.sequence < deployedSequence) {
        return "Rollback";
      }

      return "Redeploy";
    }

    const app = this.props.app;
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
      !this.props.adminConsoleMetadata?.isAirgap &&
      !this.props.adminConsoleMetadata?.isKurl;

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
  };

  getPreflightState = (version) => {
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

  renderReleaseNotes = (version) => {
    if (!version?.releaseNotes) {
      return null;
    }
    return (
      <div>
        <Icon
          icon="release-notes"
          size={24}
          onClick={() => this.props.showReleaseNotes(version?.releaseNotes)}
          data-tip="View release notes"
          className="u-marginRight--10 clickable"
        />
        <ReactTooltip effect="solid" className="replicated-tooltip" />
      </div>
    );
  };

  renderVersionAction = (version) => {
    const app = this.props.app;
    const downstream = app?.downstream;
    const { newPreflightResults } = this.props;

    let actionFn = this.props.deployVersion;
    if (this.props.isHelmManaged) {
      actionFn = () => {};
    } else if (version.needsKotsUpgrade) {
      actionFn = this.props.upgradeAdminConsole;
    } else if (version.status === "pending_download") {
      actionFn = this.props.downloadVersion;
    } else if (version.status === "failed" || version.status === "deployed") {
      actionFn = this.props.redeployVersion;
    }

    if (version.status === "pending_download") {
      let buttonText = "Download";
      if (this.props.isDownloading) {
        buttonText = "Downloading";
      } else if (version.needsKotsUpgrade) {
        buttonText = "Upgrade";
      }
      return (
        <div className="flex flex1 justifyContent--flexEnd alignItems--center">
          {this.renderReleaseNotes(version)}
          <button
            className={"btn secondary blue"}
            disabled={this.props.isDownloading}
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

    const showDeployLogs =
      isPastVersion ||
      isCurrentVersion ||
      isPendingDeployedVersion ||
      (version?.status === "superseded" && version?.status !== "pending");

    let tooltipTip;
    if (editableConfig) {
      tooltipTip = "Edit config";
    } else {
      tooltipTip = "View config";
    }

    const preflightState = this.getPreflightState(version);
    let checksStatusText;
    if (preflightState.preflightsFailed) {
      checksStatusText = "Checks failed";
    } else if (preflightState.preflightState === "warn") {
      checksStatusText = "Checks passed with warnings";
    } else {
      checksStatusText = "Checks passed";
    }

    let configScreenURL = `/app/${app.slug}/config/${version.sequence}`;
    if (this.props.isHelmManaged && version.status.startsWith("pending")) {
      configScreenURL = `${configScreenURL}?isPending=true&semver=${version.semver}`;
    }

    if (downstream.gitops?.isConnected) {
      if (version.gitDeployable === false) {
        return (
          <div
            className={
              this.props.nothingToCommit &&
              this.props.selectedDiffReleases &&
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
            {this.renderReleaseNotes(version)}
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
                    className="u-position--relative u-marginRight--10"
                    data-tip="View preflight checks"
                  >
                    <Icon
                      icon="preflight-checks"
                      size={22}
                      className="clickable"
                    />
                    {preflightState.preflightsFailed ||
                    preflightState.preflightState === "warn" ||
                    newPreflightResults ? (
                      <>
                        {preflightState.preflightsFailed ? (
                          <Icon
                            icon={"warning-circle-filled"}
                            size={12}
                            className="version-row-preflight-status-icon error-color"
                          />
                        ) : preflightState.preflightState === "warn" ? (
                          <Icon
                            icon={"warning"}
                            size={12}
                            className="version-row-preflight-status-icon warning-color"
                          />
                        ) : (
                          ""
                        )}
                        <p
                          className={`checks-running-text u-fontSize--small u-lineHeight--normal u-fontWeight--medium ${
                            preflightState.preflightsFailed
                              ? "err"
                              : preflightState.preflightState === "warn"
                              ? "warning"
                              : newPreflightResults
                              ? "success"
                              : ""
                          } 
                           ${
                             !showDeployLogs && !showActions
                               ? "without-btns"
                               : ""
                           }`}
                        >
                          {checksStatusText}
                        </p>
                      </>
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
          {this.renderReleaseNotes(version)}
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
                  className="u-position--relative u-marginRight--10"
                  data-tip="View preflight checks"
                >
                  <Icon
                    icon="preflight-checks"
                    size={20}
                    className="clickable"
                  />
                  {preflightState.preflightsFailed ||
                  preflightState.preflightState === "warn" ||
                  newPreflightResults ? (
                    <>
                      {preflightState.preflightsFailed ? (
                        <Icon
                          icon={"warning-circle-filled"}
                          size={12}
                          className="version-row-preflight-status-icon error-color"
                        />
                      ) : preflightState.preflightState === "warn" ? (
                        <Icon
                          icon={"warning"}
                          size={12}
                          className="version-row-preflight-status-icon warning-color"
                        />
                      ) : (
                        ""
                      )}
                      <p
                        className={`checks-running-text u-fontSize--small u-lineHeight--normal u-fontWeight--medium ${
                          preflightState.preflightsFailed
                            ? "err"
                            : preflightState.preflightState === "warn"
                            ? "warning"
                            : newPreflightResults
                            ? "success"
                            : ""
                        } 
                          ${
                            !showDeployLogs && !showActions
                              ? "without-btns"
                              : ""
                          }`}
                      >
                        {checksStatusText}
                      </p>
                    </>
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
        {this.renderReleaseNotes(version)}

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
                className="u-position--relative u-marginRight--10"
                data-tip="View preflight checks"
              >
                <Icon icon="preflight-checks" size={20} className="clickable" />
                {preflightState.preflightsFailed ||
                preflightState.preflightState === "warn" ||
                newPreflightResults ? (
                  <>
                    {preflightState.preflightsFailed ? (
                      <Icon
                        icon={"warning-circle-filled"}
                        size={12}
                        className="version-row-preflight-status-icon error-color"
                      />
                    ) : preflightState.preflightState === "warn" ? (
                      <Icon
                        icon={"warning"}
                        size={12}
                        className="version-row-preflight-status-icon warning-color"
                      />
                    ) : (
                      ""
                    )}
                    <p
                      className={`checks-running-text u-fontSize--small u-lineHeight--normal u-fontWeight--medium ${
                        preflightState.preflightsFailed
                          ? "err"
                          : preflightState.preflightState === "warn"
                          ? "warning"
                          : newPreflightResults
                          ? "success"
                          : ""
                      }
                      ${!showDeployLogs && !showActions ? "without-btns" : ""}
                      `}
                    >
                      {checksStatusText}
                    </p>
                  </>
                ) : null}
              </Link>
              <ReactTooltip effect="solid" className="replicated-tooltip" />
            </>
          ) : null}
        </div>
        {version.hasConfig && <EditConfigIcon version={version} />}

        {showDeployLogs ? (
          <div className="u-marginLeft--10">
            <span
              onClick={() =>
                this.props.handleViewLogs(version, version?.status === "failed")
              }
              data-tip="View deploy logs"
            >
              <Icon icon="view-logs" size={22} className="clickable" />
            </span>
            <ReactTooltip effect="solid" className="replicated-tooltip" />
            {version.status === "failed" ? (
              <div className="u-position--relative">
                <Icon icon="preflight-checks" size={20} className="clickable" />
                <Icon
                  icon={"warning-circle-filled"}
                  size={12}
                  className="version-row-preflight-status-icon error-color"
                  style={{ left: "15px", top: "-6px" }}
                />
              </div>
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
              disabled={this.isActionButtonDisabled(version)}
              onClick={() => {
                this.props.handleActionButtonClicked();
                if (needsConfiguration) {
                  this.props.history.push(configScreenURL);
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
                data-tip-disable={!this.isActionButtonDisabled(version)}
                data-tip={version.nonDeployableCause}
                data-for="disable-deployment-tooltip"
              >
                {this.deployButtonStatus(version)}
              </span>
            </button>
            <ReactTooltip effect="solid" id="disable-deployment-tooltip" />
          </div>
        )}
      </div>
    );
  };

  isActionButtonDisabled = (version) => {
    if (this.props.isHelmManaged) {
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

  renderVersionStatus = (version) => {
    const app = this.props.app;
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
                  ? `${
                      version.status === "deploying"
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
              className="link u-fontSize--small"
              onClick={() => this.props.handleViewLogs(version, true)}
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
              className="link u-fontSize--small"
              onClick={() => this.props.handleViewLogs(version, true)}
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

  render() {
    const {
      version,
      selectedDiffReleases,
      nothingToCommit,
      isChecked,
      isNew,
      gitopsEnabled,
      newPreflightResults,
      isHelmManaged,
    } = this.props;

    let showSequence = true;
    if (isHelmManaged && version.status.startsWith("pending")) {
      showSequence = false;
    }

    let sequenceLabel = "Sequence";
    if (isHelmManaged) {
      sequenceLabel = "Revision";
    }

    // Old Helm charts will not have any timestamps, so don't show current time when they are missing because it's misleading.
    let releasedTs = "";
    const tsFormat = "MM/DD/YY @ hh:mm a z";
    if (version.upstreamReleasedAt) {
      releasedTs = Utilities.dateFormat(version.upstreamReleasedAt, tsFormat);
    }

    return (
      <div
        key={version.sequence}
        className={classNames(
          `card-item VersionHistoryRowWrapper ${version.status} flex-column justifyContent--center u-padding--15`,
          {
            overlay: selectedDiffReleases,
            disabled: nothingToCommit,
            selected: isChecked && !nothingToCommit,
            "is-new": isNew,
            "show-preflight-passed-text": newPreflightResults,
          }
        )}
        style={{ minHeight: "60px" }}
        onClick={this.handleSelectReleasesToDiff}
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
            className={`${
              nothingToCommit && selectedDiffReleases && "u-opacity--half"
            } flex-column flex1 u-paddingRight--20`}
          >
            <div className="flex alignItems--center">
              <p className="u-fontSize--header2 u-fontWeight--bold u-lineHeight--medium card-item-title">
                {version.versionLabel || version.title}
              </p>
              {showSequence && (
                <p
                  className="u-fontSize--small u-textColor--bodyCopy u-fontWeight--medium u-marginLeft--10"
                  style={{ marginTop: "2px" }}
                >
                  {sequenceLabel} {version.sequence}
                </p>
              )}
              {version.isRequired && (
                <span className="status-tag required u-marginLeft--10">
                  {" "}
                  Required{" "}
                </span>
              )}
            </div>
            {releasedTs && (
              <p className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy u-marginTop--5">
                {" "}
                Released{" "}
                <span className="u-fontWeight--bold">{releasedTs}</span>
              </p>
            )}
            {this.renderDiff(version)}
            {version.yamlErrors && (
              <YamlErrors
                yamlErrors={version.yamlErrors}
                handleShowDetailsClicked={() =>
                  this.props.toggleShowDetailsModal(
                    version.yamlErrors,
                    version.sequence
                  )
                }
              />
            )}
          </div>
          <div
            className={`${
              nothingToCommit && selectedDiffReleases && "u-opacity--half"
            } flex-column flex1 justifyContent--center`}
          >
            <p className="u-fontSize--small u-fontWeight--bold u-textColor--lightAccent u-lineHeight--default">
              {version.source}
            </p>
            {gitopsEnabled && version.status !== "pending_download" ? null : (
              <div className="flex flex-auto u-marginTop--10">
                {this.renderVersionStatus(version)}
              </div>
            )}
          </div>
          <div
            className={`${
              nothingToCommit && selectedDiffReleases && "u-opacity--half"
            } flex-column flex-auto alignItems--flexEnd justifyContent--center`}
          >
            {this.renderVersionAction(version)}
          </div>
        </div>
        {this.props.renderVersionDownloadStatus(version)}
      </div>
    );
  }
}

export { AppVersionHistoryRow };
