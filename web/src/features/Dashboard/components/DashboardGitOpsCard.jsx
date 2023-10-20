import { Link } from "react-router-dom";
import Loader from "@src/components/shared/Loader";
import { getReadableGitOpsProviderName } from "@src/utilities/utilities";
import Icon from "@src/components/Icon";

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
        <p className="card-title flex alignItems--center">
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
              <Icon
                icon="check-update"
                size={18}
                className="clickable u-marginRight--5"
              />
              <span
                className="link u-fontSize--small"
                onClick={onCheckForUpdates}
              >
                Check for update
              </span>
            </div>
          )}
          <Icon
            icon="schedule-sync"
            size={16}
            className="clickable u-marginRight--5"
          />
          <span
            className="link u-fontSize--small u-lineHeight--default"
            onClick={showAutomaticUpdatesModal}
          >
            Configure automatic updates
          </span>
        </div>
      </div>
      <div className="card-item">
        <p className="u-fontSize--normal u-fontWeight--medium u-textColor--header u-lineHeight--normal">
          GitOps is enabled for this application. Visit{" "}
          <a
            target="_blank"
            rel="noopener noreferrer"
            href={gitops.uri}
            className="link"
          >
            {isAirgap
              ? gitops.uri
              : getReadableGitOpsProviderName(gitops.provider)}
          </a>{" "}
          to track all versions and to view information about the currently
          deployed version. Config for the latest version can be edited from the{" "}
          <Link
            to={`/app/${appSlug}/config/${latestConfigSequence}`}
            className="link"
          >
            Config
          </Link>{" "}
          page in the admin console.
        </p>
      </div>
      <div className="u-marginTop--10">
        <Link to="/gitops" className="link  u-fontSize--small">
          Manage GitOps settings
          <Icon
            icon="next-arrow"
            size={10}
            className="has-arrow u-marginLeft--5"
          />
        </Link>
      </div>
    </div>
  );
}
