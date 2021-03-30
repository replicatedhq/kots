
import React from "react";
import { Link } from "react-router-dom";
import ReactTooltip from "react-tooltip"

import MountAware from "../shared/MountAware";
import Loader from "../shared/Loader";
import { Utilities } from "../../utilities/utilities";


function getCurrentVersionStatus(version, viewLogs) {
  if (version?.status === "deployed" || version?.status === "merged" || version?.status === "pending") {
    return <span className="u-fontSize--small u-lineHeight--normal u-color--dustyGray u-fontWeight--medium flex alignItems--center u-marginTop--8"> <span className="icon checkmark-icon u-marginRight--5" /> {Utilities.toTitleCase(version?.status).replace("_", " ")} </span>
  } else if (version?.status === "failed") {
    return <span className="u-fontSize--small u-lineHeight--normal u-color--red u-fontWeight--medium flex alignItems--center u-marginTop--8"> <span className="icon error-small u-marginRight--5" /> Failed <span className="u-marginLeft--5 replicated-link u-fontSize--small" onClick={() => viewLogs(version, true)}> See details </span></span>
  } else if (version?.status === "deploying") {
    return (
      <span className="flex alignItems--center u-fontSize--small u-lineHeight--normal u-color--dustyGray u-fontWeight--medium u-marginTop--8">
        <Loader className="flex alignItems--center u-marginRight--5" size="16" />
            Deploying
      </span>);
  } else {
    return <span className="u-fontSize--small u-lineHeight--normal u-color--dustyGray u-fontWeight--medium flex alignItems--center u-marginTop--8"> {Utilities.toTitleCase(version?.status).replace("_", " ")} </span>
  }
}

export default function AppVersionHistoryHeader(props) {
  const { app, currentDownstreamVersion, showDownstreamReleaseNotes, slug, handleViewLogs, onCheckForUpdates, showUpdateCheckerModal,
    checkingForUpdates, isBundleUploading, airgapUploader, pendingVersions, showOnlineUI, showAirgapUI, updateText, noUpdateAvailiableMsg } = props;

  return (
    <div className="flex flex-auto alignItems--center justifyContent--center u-marginTop--10 u-marginBottom--30">
      <div className="upstream-version-box-wrapper flex flex1">
        <div className="flex flex1">
          {app.iconUri &&
            <div className="flex-auto u-marginRight--10">
              <div className="watch-icon" style={{ backgroundImage: `url(${app.iconUri})` }}></div>
            </div>
          }
          <div className="flex1 flex-column">
            <p className="u-fontSize--small u-fontWeight--bold u-lineHeight--normal u-color--tuna"> {currentDownstreamVersion?.versionLabel ? "Current version" : "No current version deployed"} </p>
            <div className="flex alignItems--center u-marginTop--5">
              <p className="u-fontSize--header2 u-fontWeight--bold u-color--tuna"> {currentDownstreamVersion ? currentDownstreamVersion.versionLabel : "---"}</p>
              <p className="u-fontSize--small u-lineHeight--normal u-color--tundora u-fontWeight--medium u-marginLeft--10"> {currentDownstreamVersion ? `Sequence ${currentDownstreamVersion?.sequence}` : null}</p>
            </div>
            {currentDownstreamVersion?.deployedAt ? <p className="u-fontSize--small u-lineHeight--normal u-color--silverSand u-fontWeight--medium u-marginTop--5">{`${Utilities.dateFormat(currentDownstreamVersion.deployedAt, "MMMM D, YYYY @ hh:mm a z")}`}</p> : null}
            {currentDownstreamVersion && getCurrentVersionStatus(currentDownstreamVersion, handleViewLogs)}
            {currentDownstreamVersion ?
              <div className="flex alignItems--center u-marginTop--8 u-marginTop--8">
                {currentDownstreamVersion?.releaseNotes &&
                  <div>
                    <span className="icon releaseNotes--icon u-marginRight--10 u-cursor--pointer" onClick={() => showDownstreamReleaseNotes(currentDownstreamVersion?.releaseNotes)} data-tip="View release notes" />
                    <ReactTooltip effect="solid" className="replicated-tooltip" />
                  </div>}
                <div>
                  <Link to={`/app/${slug}/downstreams/${app.downstreams[0].cluster?.slug}/version-history/preflight/${currentDownstreamVersion?.sequence}`}
                    className="icon preflightChecks--icon u-marginRight--10 u-cursor--pointer"
                    data-tip="View preflight checks" />
                  <ReactTooltip effect="solid" className="replicated-tooltip" />
                </div>
                <div>
                  <span className="icon deployLogs--icon u-marginRight--10 u-cursor--pointer" onClick={() => handleViewLogs(currentDownstreamVersion, currentDownstreamVersion?.status === "failed")} data-tip="View deploy logs" />
                  <ReactTooltip effect="solid" className="replicated-tooltip" />
                </div>
                {app.isConfigurable &&
                  <div>
                    <Link to={`/app/${slug}/config/${app?.downstreams[0]?.currentVersion?.parentSequence}`} className="icon config--icon u-cursor--pointer" data-tip={`${Utilities.checkIsDeployedConfigLatest(app) ? "Edit config" : "View config"}`} />
                    <ReactTooltip effect="solid" className="replicated-tooltip" />
                  </div>}
              </div> : null}
          </div>
        </div>
        {!app.cluster &&
          <div className={`flex flex1 justifyContent--center ${checkingForUpdates && !isBundleUploading && "alignItems--center"}`}>
            {checkingForUpdates && !isBundleUploading
              ? <Loader size="32" />
              : showAirgapUI ?
                <div className="flex flex-column justifyContent--center">
                  <p className="u-fontSize--small u-fontWeight--bold u-lineHeight--normal u-color--tuna"> Upload a new version</p>
                  <span className="u-fontSize--small u-lineHeight--normal u-color--dustyGray u-marginTop--8"> When you have an Airgap Bundle for a new version, you can upload that new version here. </span>
                  {airgapUploader ?
                    <MountAware className="flex alignItems--center" id="bundle-dropzone" onMount={el => airgapUploader.assignElement(el)}>
                      <span className="btn primary blue u-marginTop--10">Upload a new version</span>
                    </MountAware>
                    : null
                  }
                </div>
                : showOnlineUI ?
                  <div className="flex1 flex-column justifyContent--center">
                    {pendingVersions?.length > 0 ?
                      <div className="flex flex-column">
                        <p className="u-fontSize--small u-lineHeight--normal u-color--chateauGreen u-fontWeight--bold">New version available</p>
                        <div className="flex flex-column u-marginTop--5 new-version-wrapper">
                          <div className="flex flex1 alignItems--center">
                            <span className="u-fontSize--larger u-lineHeight--medium u-fontWeight--bold u-color--tundora">{pendingVersions[0]?.versionLabel}</span>
                            <span className="u-fontSize--small u-lineHeight--normal u-fontWeight--medium u-color--tundora u-marginLeft--5"> Sequence {pendingVersions[0]?.sequence}</span>
                          </div>
                          <div className="flex flex1 alignItems--center">
                            {pendingVersions[0]?.createdOn || pendingVersions[0].upstreamReleasedAt ?
                              <p className="u-fontSize--small u-lineHeight--normal u-fontWeight--medium u-color--dustyGray">Released <span className="u-fontWeight--bold">{pendingVersions[0].upstreamReleasedAt ? Utilities.dateFormat(pendingVersions[0]?.upstreamReleasedAt, "MMMM D, YYYY") : Utilities.dateFormat(pendingVersions[0]?.createdOn, "MMMM D, YYYY")}</span></p>
                              : null}
                            {pendingVersions[0]?.releaseNotes ? <span className="release-notes-link u-fontWeight--medium u-fontSize--small u-fontWeight--medium u-marginLeft--10 flex alignItems--center" onClick={() => showDownstreamReleaseNotes(pendingVersions[0]?.releaseNotes)}><span className="icon releaseNotes-small--icon clickable u-marginRight--5" />Release notes</span> : null}
                          </div>
                        </div>
                      </div>
                      : <p className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal" style={{ color: "#C4C8CA" }}> No new version available </p>}
                    <div className="flex alignItems--center u-marginTop--10">
                      <button className="btn primary blue" onClick={onCheckForUpdates}>Check for update</button>
                      <span className="icon settings-small-icon u-marginLeft--10 u-cursor--pointer" onClick={showUpdateCheckerModal} data-tip="Configure automatic update checks"></span>
                      <ReactTooltip effect="solid" className="replicated-tooltip" />
                    </div>
                    {updateText}
                    {noUpdateAvailiableMsg}
                  </div>
                  : null
            }
            {!showOnlineUI && updateText}
            {!showOnlineUI && noUpdateAvailiableMsg}
          </div>
        }
      </div>
    </div>
  )
}
