import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import find from "lodash/find";
import classNames from "classnames";
import ReactTooltip from "react-tooltip";

import Loader from "../../components/shared/Loader";

import { Utilities, getPreflightResultState } from "@src/utilities/utilities";

import { YamlErrors } from "./YamlErrors";
import Icon from "@src/components/Icon";

import { ViewDiffButton } from "@features/VersionDiff/ViewDiffButton";
import { Metadata, Version, VersionDownloadStatus } from "@types";
import { useSelectedApp } from "@features/App/hooks/useSelectedApp";
import PreflightIcon from "@features/App/PreflightIcon";

interface Props {
  index: number;
  adminConsoleMetadata: Metadata;
  isEmbeddedCluster: boolean;
  deployVersion: (version: Version) => void;
  downloadVersion: (version: Version) => void;
  gitopsEnabled: boolean;
  handleSelectReleasesToDiff: (version: Version, isChecked: boolean) => void;
  handleViewLogs: (version: Version | null, isFailing: boolean) => void;
  isChecked: boolean;
  isDownloading: boolean;
  isNew: boolean;
  navigate: ReturnType<typeof useNavigate>;
  newPreflightResults: boolean;
  nothingToCommit: boolean;
  onWhyNoGeneratedDiffClicked: (rowVersion: Version) => void;
  onWhyUnableToGeneratedDiffClicked: (rowVersion: Version) => void;
  onViewDiffClicked: (firstSequence: number, secondSequence: number) => void;
  redeployVersion: (version: Version) => void;
  selectedDiffReleases: boolean;
  showReleaseNotes: (releaseNotes: string) => void;
  showVersionPreviousDownloadStatus: boolean;
  showVersionDownloadingStatus: boolean;
  toggleShowDetailsModal: (
    yamlErrorDetails: string[],
    selectedSequence: number
  ) => void;
  upgradeAdminConsole: (version: Version) => void;
  version: Version;
  versionDownloadStatus: VersionDownloadStatus;
  versionHistory: Version[];
}

function AppVersionHistoryRow(props: Props) {
  // TODO: move this into a selector
  const [showViewDiffButton, setShowViewDiffButton] = useState(
    !props.version.source?.includes("Airgap Install") &&
      !props.version.source?.includes("Online Install")
  );

  const selectedApp = useSelectedApp();

  useEffect(() => {
    setShowViewDiffButton(
      !props.version.source?.includes("Airgap Install") &&
        !props.version.source?.includes("Online Install")
    );
  }, [props.version.source]);

  const handleSelectReleasesToDiff = () => {
    if (!props.selectedDiffReleases) {
      return;
    }
    if (props.nothingToCommit) {
      return;
    }
    props.handleSelectReleasesToDiff(props.version, !props.isChecked);
  };

  const deployButtonStatus = (version: Version) => {
    const downstream = selectedApp?.downstream;

    const isCurrentVersion =
      version.sequence === downstream?.currentVersion?.sequence;
    const isDeploying = version.status === "deploying";
    const isPastVersion = find(downstream?.pastVersions, {
      sequence: version.sequence,
    });
    const needsConfiguration = version.status === "pending_config";
    const isRollback =
      isPastVersion && version.deployedAt && selectedApp?.allowRollback;
    const isRedeploy =
      isCurrentVersion &&
      (version.status === "failed" || version.status === "deployed");
    const canUpdateKots =
      version.needsKotsUpgrade &&
      !props.adminConsoleMetadata?.isAirgap &&
      !props.adminConsoleMetadata?.isKurl;

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

  const getPreflightState = (version: Version) => {
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

  const renderReleaseNotes = (version: Version) => {
    if (!version?.releaseNotes) {
      return null;
    }
    return (
      <div>
        <Icon
          icon="release-notes"
          size={24}
          onClick={() => props.showReleaseNotes(version?.releaseNotes)}
          data-tip="View release notes"
          className="u-marginRight--10 clickable"
          data-testid="release-notes-icon"
        />
        <ReactTooltip effect="solid" className="replicated-tooltip" />
      </div>
    );
  };

  const isActionButtonDisabled = (version: Version) => {
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

  const renderVersionAction = (version: Version) => {
    const app = selectedApp;
    const downstream = app?.downstream;
    const { newPreflightResults } = props;

    // useDeployAppVersion
    let actionFn = props.deployVersion;
    if (version.needsKotsUpgrade) {
      // postUpdateAdminConsole
      actionFn = props.upgradeAdminConsole;
    } else if (version.status === "pending_download") {
      // postDownloadVersion
      actionFn = props.downloadVersion;
    } else if (version.status === "failed" || version.status === "deployed") {
      // postRedeployVersion
      actionFn = props.redeployVersion;
    }

    if (version.status === "pending_download") {
      let buttonText = "Download";
      if (props.isDownloading) {
        buttonText = "Downloading";
      } else if (version.needsKotsUpgrade) {
        buttonText = "Upgrade";
      }
      return (
        <div className="flex flex1 justifyContent--flexEnd alignItems--center">
          {renderReleaseNotes(version)}
          <button
            className={"btn secondary blue"}
            disabled={props.isDownloading}
            onClick={() => actionFn(version)}
            data-testid="version-action-button"
          >
            {buttonText}
          </button>
        </div>
      );
    }

    const isCurrentVersion =
      version.sequence === downstream?.currentVersion?.sequence;
    const isPastVersion = find(downstream?.pastVersions, {
      sequence: version.sequence,
    });
    const isPendingDeployedVersion = find(downstream?.pendingVersions, {
      sequence: version.sequence,
      status: "deployed",
    });
    const needsConfiguration = version.status === "pending_config";
    const showActions = !isPastVersion || selectedApp?.allowRollback;
    const isRedeploy =
      isCurrentVersion &&
      (version.status === "failed" || version.status === "deployed");
    const isRollback =
      isPastVersion && version.deployedAt && selectedApp?.allowRollback;

    const isSecondaryBtn =
      isPastVersion || needsConfiguration || (isRedeploy && !isRollback);
    const isPrimaryButton = !isSecondaryBtn && !isRedeploy && !isRollback;

    const showDeployLogs =
      (isPastVersion || isCurrentVersion || isPendingDeployedVersion) &&
      version?.status !== "pending";

    // in embedded cluster, past versions cannot be edited
    const isConfigReadOnly = props.isEmbeddedCluster && isPastVersion;
    let configTooltip;
    if (isConfigReadOnly) {
      configTooltip = "View config";
    } else {
      configTooltip = "Edit config";
    }

    const preflightState = getPreflightState(version);

    const configScreenURL = `/app/${selectedApp?.slug}/config/${version.sequence}`;

    // CONNECTED TO GITOPS //
    if (downstream?.gitops?.isConnected) {
      if (version.gitDeployable === false) {
        return (
          <div
            className={
              props.nothingToCommit && props.selectedDiffReleases
                ? "u-opacity--half"
                : ""
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
                <div className="u-position--relative" data-testid="preflight-checks-loader-gitops">
                  <Loader size="30" />
                  <p className="checks-running-text u-fontSize--small u-lineHeight--normal u-fontWeight--medium">
                    Running checks
                  </p>
                </div>
              ) : preflightState.preflightState !== "" ? (
                <>
                  <PreflightIcon
                    app={app}
                    version={version}
                    showDeployLogs={showDeployLogs}
                    showActions={showActions}
                    preflightState={preflightState}
                    showText={true}
                    className={"tw-mr-2"}
                    newPreflightResults={newPreflightResults}
                  />
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
              <div className="u-position--relative" data-testid="preflight-checks-loader-gitops">
                <Loader size="30" />
                <p className="checks-running-text u-fontSize--small u-lineHeight--normal u-fontWeight--medium">
                  Running checks
                </p>
              </div>
            ) : preflightState.preflightState !== "" ? (
              <>
                <PreflightIcon
                  app={app}
                  version={version}
                  showDeployLogs={showDeployLogs}
                  showText={true}
                  showActions={showActions}
                  preflightState={preflightState}
                  className={"tw-mr-2"}
                  newPreflightResults={newPreflightResults}
                />
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
    // END OF CONNECTED TO GITOPS //

    return (
      <div className="flex flex1 justifyContent--flexEnd alignItems--center">
        {renderReleaseNotes(version)}
        <div>
          {version.status === "pending_preflight" ? (
            <div className="u-marginRight--10 u-position--relative" data-testid="preflight-checks-loader">
              <Loader size="30" />
              <p className="checks-running-text u-fontSize--small u-lineHeight--normal u-fontWeight--medium">
                Running checks
              </p>
            </div>
          ) : preflightState.preflightState !== "" ? (
            <>
              <PreflightIcon
                app={app}
                showText={true}
                version={version}
                showDeployLogs={showDeployLogs}
                showActions={showActions}
                preflightState={preflightState}
                className={"tw-mr-2"}
                newPreflightResults={newPreflightResults}
              />
              <ReactTooltip effect="solid" className="replicated-tooltip" />
            </>
          ) : null}
        </div>
        {version.hasConfig && (
          <div className="flex alignItems--center">
            <Link to={configScreenURL} data-tip={configTooltip}>
              <Icon
                icon={isConfigReadOnly ? "view-config" : "edit-config"}
                size={22}
              />
            </Link>
            <ReactTooltip effect="solid" className="replicated-tooltip" />
          </div>
        )}
        {showDeployLogs ? (
          <div className="u-marginLeft--10">
            <span
              onClick={() =>
                props.handleViewLogs(version, version?.status === "failed")
              }
              data-tip="View deploy logs"
            >
              <Icon icon="view-logs" size={22} className="clickable" />
            </span>
            <ReactTooltip effect="solid" className="replicated-tooltip" />
          </div>
        ) : null}
        {showActions && (
          <div className="flex alignItems--center">
            <button
              className={classNames("btn u-marginLeft--10", {
                "secondary blue": isSecondaryBtn || isRollback,
                "primary blue": isPrimaryButton,
              })}
              disabled={isActionButtonDisabled(version)}
              onClick={() => {
                if (needsConfiguration) {
                  props?.navigate(configScreenURL);
                  return null;
                }
                if (isRollback) {
                  actionFn(version);
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

  const renderVersionStatus = (version: Version) => {
    const app = selectedApp;
    const downstream = app?.downstream;
    if (!downstream) {
      return null;
    }

    const isPastVersion = find(downstream?.pastVersions, {
      sequence: version.sequence,
    });
    const isPendingDeployedVersion = find(downstream?.pendingVersions, {
      sequence: version.sequence,
      status: "deployed",
    });

    if (!isPastVersion && !isPendingDeployedVersion) {
      if (version.status === "deployed") {
        return (
          <div>
            <span
              className="status-tag success flex-auto u-cursor--default"
              data-tip={
                version.deployedAt
                  ? `${"Deployed"} ${Utilities.dateFormat(
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
              onClick={() => props.handleViewLogs(version, true)}
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
      if (version.status === "deployed") {
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
              onClick={() => props.handleViewLogs(version, true)}
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

  const {
    index,
    version,
    selectedDiffReleases,
    nothingToCommit,
    isChecked,
    isNew,
    gitopsEnabled,
    newPreflightResults,
  } = props;

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
      onClick={handleSelectReleasesToDiff}
      data-testid={`version-history-row-${index}`}
    >
      <>
        <div className="VersionHistoryRow flex flex-auto">
          {selectedDiffReleases && (
            <div
              className={classNames(
                "checkbox u-marginRight--20",
                { checked: isChecked && !nothingToCommit },
                { disabled: nothingToCommit }
              )}
              data-testid="diff-checkbox"
            />
          )}
          <div
            className={`${
              nothingToCommit && selectedDiffReleases && "u-opacity--half"
            } flex-column flex1 u-paddingRight--20`}
          >
            <div className="flex alignItems--center">
              <p
                className="u-fontSize--header2 u-fontWeight--bold u-lineHeight--medium card-item-title"
                data-testid="version-label"
              >
                {version.versionLabel || version.title}
              </p>

              {version.isRequired && (
                <span className="status-tag required u-marginLeft--10">
                  {" "}
                  Required{" "}
                </span>
              )}
            </div>{" "}
            <p
              className="u-fontSize--small u-textColor--bodyCopy u-fontWeight--medium"
              style={{ marginTop: "2px" }}
              data-testid="version-sequence"
            >
              Sequence {version.sequence}
            </p>
            {releasedTs && (
              <p className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy u-marginTop--5">
                {" "}
                Released{" "}
                <span className="u-fontWeight--bold">{releasedTs}</span>
              </p>
            )}
            {showViewDiffButton && (
              <ViewDiffButton
                onWhyNoGeneratedDiffClicked={props.onWhyNoGeneratedDiffClicked}
                onWhyUnableToGeneratedDiffClicked={
                  props.onWhyUnableToGeneratedDiffClicked
                }
                onViewDiffClicked={(firstSequence, secondSequence) =>
                  props.onViewDiffClicked(firstSequence, secondSequence)
                }
                version={props.version}
                versionHistory={props.versionHistory}
              />
            )}
            {version.yamlErrors && (
              <YamlErrors
                yamlErrors={version.yamlErrors}
                handleShowDetailsClicked={() =>
                  props.toggleShowDetailsModal(
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
              <div
                className="flex flex-auto u-marginTop--10"
                data-testid="version-status"
              >
                {renderVersionStatus(version)}
              </div>
            )}
          </div>
          <div
            className={`${
              nothingToCommit && selectedDiffReleases && "u-opacity--half"
            } flex-column flex-auto alignItems--flexEnd justifyContent--center`}
          >
            {renderVersionAction(version)}
          </div>
        </div>
        {props.showVersionPreviousDownloadStatus && (
          <div className="flex alignItems--center justifyContent--flexEnd">
            <span
              className={`u-textColor--bodyCopy u-fontWeight--medium u-fontSize--small u-lineHeight--default ${
                version.downloadStatus.status === "failed"
                  ? "u-textColor--error"
                  : ""
              }`}
              data-testid="version-download-status"
            >
              {version.downloadStatus.message}
            </span>
          </div>
        )}
        {props.showVersionDownloadingStatus && (
          <div
            className="flex alignItems--center justifyContent--flexEnd"
            data-testid="version-downloading-status"
          >
            {props.versionDownloadStatus?.downloadingVersion && (
              <Loader className="u-marginRight--5" size="15" />
            )}
            <span
              className={`u-textColor--bodyCopy u-fontWeight--medium u-fontSize--small u-lineHeight--default ${
                props.versionDownloadStatus?.downloadingVersionError
                  ? "u-textColor--error"
                  : ""
              }`}
            >
              {props.versionDownloadStatus?.downloadingVersionMessage
                ? props.versionDownloadStatus?.downloadingVersionMessage
                : props.versionDownloadStatus?.downloadingVersion
                ? "Downloading"
                : ""}
            </span>
          </div>
        )}
      </>
    </div>
  );
}

export { AppVersionHistoryRow };
