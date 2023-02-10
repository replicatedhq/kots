import React from "react";
import { Version } from "@types";
import { AppVersionHistoryRow } from "./AppVersionHistoryRow.tsx";
import { useDownloadValues } from "./api";
import { useSelectedApp } from "@features/App/hooks/useSelectedApp";
import { useIsHelmManaged } from "@components/hooks";
// TODO: get rid of this
// new Date(new Date().getTime() - 10*1000)
import {
  secondsAgo,
} from "../../utilities/utilities";

// renderAppVersionHistoryRow
function AppVersionHistoryCard({
  isChecked,
  selectedDiffReleases,
  version
}: {
  isChecked: boolean;
  selectedDiffReleases: boolean;
  version: Version
}) {
  const { selectedApp } = useSelectedApp();
  const isHelmManaged = useIsHelmManaged();
  const { clearError: clearDownloadError, download, error: downloadError } = useDownloadValues({
    appSlug: selectedApp?.slug,
    fileName: "values.yaml",
    sequence: version.parentSequence,
    versionLabel: version.versionLabel,
    isPending: version.status.startsWith("pending") && isHelmManaged,
  });

  // TODO: invert this- shouldn't be in this component
  if (
    !version ||
    Object.keys(version).length === 0 ||
    (selectedDiffReleases && version.status === "pending_download")
  ) {
    // non-downloaded versions can't be diffed
    return null;
  }

  // TODO: move this outside of the component and figure out what it means
  // const isChecked = !!this.state.checkedReleasesToDiff.find(
  //   (diffRelease) => diffRelease.parentSequence === version.parentSequence
  // );

  /* tasks

  - convert AppVersionHistoryRow to a functional component
  - clean up these props
  - add a modal provider and move the helm deploy modal out of this component
  - maybe just get rid of this component altogether and use AppVersionHistoryRow directly

  */

  return (
    <>
      <AppVersionHistoryRow
        handleActionButtonClicked={() =>
          this.handleActionButtonClicked(
            version.versionLabel,
            version.sequence
          )
        }
        isHelmManaged={isHelmManaged}
        key={version.sequence}
        app={selectedApp}
        version={version}
        selectedDiffReleases={this.state.selectedDiffReleases}
        nothingToCommit={selectedApp?.downstream?.gitops?.isConnected && !version.commitUrl}
        isChecked={isChecked}
        isNew={secondsAgo(version.createdOn) < 10}
        newPreflightResults={version.preflightResultCreatedAt && secondsAgo(version.preflightResultCreatedAt) < 12 ? true : false}
        showReleaseNotes={this.showReleaseNotes}
        renderDiff={this.renderDiff}
        toggleShowDetailsModal={this.toggleShowDetailsModal}
        gitopsEnabled={selectedApp?.downstream?.gitops?.isConnected}
        deployVersion={this.deployVersion}
        redeployVersion={this.redeployVersion}
        downloadVersion={this.downloadVersion}
        upgradeAdminConsole={this.upgradeAdminConsole}
        handleViewLogs={this.handleViewLogs}
        handleSelectReleasesToDiff={this.handleSelectReleasesToDiff}
        renderVersionDownloadStatus={this.renderVersionDownloadStatus}
        isDownloading={
          this.state.versionDownloadStatuses?.[version.sequence]
            ?.downloadingVersion
        }
        adminConsoleMetadata={this.props.adminConsoleMetadata}
      />
      {this.state.showHelmDeployModalForVersionLabel ===
        version.versionLabel &&
        this.state.showHelmDeployModalForSequence === version.sequence && (

          <>
            <HelmDeployModal
              appSlug={this.props?.app?.slug}
              chartPath={this.props?.app?.chartPath || ""}
              downloadClicked={download}
              downloadError={downloadError}
              //isDownloading={isDownloading}
              hideHelmDeployModal={() => {
                this.setState({
                  showHelmDeployModalForVersionLabel: "",
                });
                clearDownloadError();
              }}
              registryUsername={this.props?.app?.credentials?.username}
              registryPassword={this.props?.app?.credentials?.password}
              revision={
                this.deployButtonStatus(version) === "Rollback"
                  ? version.sequence
                  : null
              }
              showHelmDeployModal={true}
              showDownloadValues={
                this.deployButtonStatus(version) === "Deploy"
              }
              subtitle={
                this.deployButtonStatus(version) === "Rollback"
                  ? `Follow the steps below to rollback to revision ${version.sequence}.`
                  : this.deployButtonStatus(version) === "Redeploy"
                    ? "Follow the steps below to redeploy the release using the currently deployed chart version and values."
                    : "Follow the steps below to upgrade the release."
              }
              title={` ${this.deployButtonStatus(version)} ${this.props?.app.slug
                } ${this.deployButtonStatus(version) === "Deploy"
                  ? version.versionLabel
                  : ""
                }`}
              upgradeTitle={
                this.deployButtonStatus(version) === "Rollback"
                  ? "Rollback release"
                  : this.deployButtonStatus(version) === "Redeploy"
                    ? "Redeploy release"
                    : "Upgrade release"
              }
              version={version.versionLabel}
              namespace={this.props?.app?.namespace}
            />
            <a
              href={url}
              download={name}
              className="hidden"
              ref={ref}
            />
          </>
        )}
    </>
  );
};

return (
  <div> </div>);
}

export { AppVersionHistoryCard };