import React from "react";
import { Link } from "react-router-dom";
import Loader from "../shared/Loader";
import { getReadableGitOpsProviderName } from "../../utilities/utilities";

export default function DashboardGitOpsCard(props) {
  const {
    gitops,
    isAirgap,
    appSlug,
    checkingForUpdates,
    latestConfigSequence,
    isBundleUploading,
    checkingUpdateText,
    checkingUpdateTextShort,
    noUpdatesAvalable,
    onCheckForUpdates,
    showAutomaticUpdatesModal,
  } = props;

  if (!gitops) {
    return null;
  }

  return (
    <div className="dashboard-card gitops">
      <div className="flex flex1 justifyContent--spaceBetween alignItems--center u-marginBottom--10">
        <p className="u-fontSize--large u-textColor--primary u-fontWeight--bold flex alignItems--center">
          <span
            className={`icon gitopsService--${gitops.provider} u-marginRight--10`}
          />
          GitOps Enabled
        </p>
        <div className="flex alignItems--center">
          {checkingForUpdates && !isBundleUploading ? (
            <div className="flex alignItems--center u-marginRight--20">
              <Loader className="u-marginRight--5" size="15" />
              <span className="u-textColor--bodyCopy u-fontWeight--medium u-fontSize--small u-lineHeight--default">
                {checkingUpdateText === ""
                  ? "Checking for updates"
                  : checkingUpdateTextShort}
              </span>
            </div>
          ) : noUpdatesAvalable ? (
            <div className="flex alignItems--center u-marginRight--20">
              <span className="u-textColor--primary u-fontWeight--medium u-fontSize--small u-lineHeight--default">
                Already up to date
              </span>
            </div>
          ) : (
            <div className="flex alignItems--center u-marginRight--20">
              <span className="icon clickable dashboard-card-check-update-icon u-marginRight--5" />
              <span
                className="replicated-link u-fontSize--small"
                onClick={onCheckForUpdates}
              >
                Check for update
              </span>
            </div>
          )}
          <span className="icon clickable dashboard-card-configure-update-icon u-marginRight--5" />
          <span
            className="replicated-link u-fontSize--small u-lineHeight--default"
            onClick={showAutomaticUpdatesModal}
          >
            Configure automatic updates
          </span>
        </div>
      </div>
      <div className="VersionCard-content--wrapper">
        <p className="u-fontSize--normal u-fontWeight--medium u-textColor--header u-lineHeight--normal">
          GitOps is enabled for this application. Visit{" "}
          <a
            target="_blank"
            rel="noopener noreferrer"
            href={gitops.uri}
            className="replicated-link"
          >
            {isAirgap
              ? gitops.uri
              : getReadableGitOpsProviderName(gitops.provider)}
          </a>{" "}
          to track all versions and to view information about the currently
          deployed version. Config for the latest version can be edited from the{" "}
          <Link
            to={`/app/${appSlug}/config/${latestConfigSequence}`}
            className="replicated-link"
          >
            Config
          </Link>{" "}
          page in the admin console.
        </p>
      </div>
      <div className="u-marginTop--10">
        <Link
          to="/gitops"
          className="replicated-link has-arrow u-fontSize--small"
        >
          Manage GitOps settings
        </Link>
      </div>
    </div>
  );
}
