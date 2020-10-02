
import React from "react";
import dayjs from "dayjs";
import classNames from "classnames";
import isSameOrAfter from "dayjs/plugin/isSameOrAfter";
dayjs.extend(isSameOrAfter);


export default function AppVersionHistoryRow(props) {
  const { version, selectedDiffReleases, nothingToCommit,
    isChecked, isNew, showDownstreamReleaseNotes, renderSourceAndDiff,
    yamlErrorsDetails, renderYamlErrors, renderVersionStatus, renderViewPreflights, renderVersionAction, gitopsEnabled } = props;

  return (
    <div key={version.sequence}
      className={classNames(`VersionHistoryDeploymentRow ${version.status} flex flex-auto`, { "overlay": selectedDiffReleases, "disabled": nothingToCommit, "selected": (isChecked && !nothingToCommit), "is-new": isNew })}
      onClick={() => selectedDiffReleases && !nothingToCommit && this.handleSelectReleasesToDiff(version, !isChecked)}
    >
      {selectedDiffReleases && <div className={classNames("checkbox u-marginRight--20", { "checked": (isChecked && !nothingToCommit) }, { "disabled": nothingToCommit })} />}
      <div className={`${nothingToCommit && selectedDiffReleases && "u-opacity--half"} flex-column flex1 u-paddingRight--20`}>
        <div className="flex alignItems--center">
          <p className="u-fontSize--large u-fontWeight--bold u-lineHeight--medium u-color--tuna">{version.versionLabel || version.title}</p>
          <p className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-color--tundora u-marginLeft--5" style={{ marginTop: "2px" }}>Sequence {version.sequence}</p>
        </div>
        <div className="flex alignItems--center u-marginTop--10"></div>
        <div className="flex flex1 u-marginTop--15 alignItems--center">
          <p className="u-fontSize--small u-lineHeight--normal u-color--dustyGray u-fontWeight--medium">Released <span className="u-fontWeight--bold">{dayjs(version.createdOn).format("MMMM D, YYYY")}</span></p>
          {version.releaseNotes ?
            <p className="release-notes-link u-fontSize--small u-lineHeight--normal u-marginLeft--5 flex alignItems--center" onClick={() => showDownstreamReleaseNotes(version.releaseNotes)}> <span className="icon releaseNotes-small--icon u-marginRight--5" />Release notes</p> : null}
        </div>
      </div>
      <div className={`${nothingToCommit && selectedDiffReleases && "u-opacity--half"} flex-column flex1`}>
        <div className="flex flex-column">
          <p className="u-fontSize--normal u-fontWeight--bold u-color--tuna">{version.source}</p>
          <div className="flex alignItems--center u-fontSize--small u-marginTop--10 u-color--dustyGray">
            {renderSourceAndDiff(version)}
            {yamlErrorsDetails && renderYamlErrors(yamlErrorsDetails, version)}
          </div>
        </div>
        <div className="flex flex1 alignItems--flexEnd"> {gitopsEnabled ? renderViewPreflights(version) : renderVersionStatus(version)}</div>
      </div>
      <div className={`${nothingToCommit && selectedDiffReleases && "u-opacity--half"} flex-column flex1 alignItems--flexEnd`}>
        <div>
          {renderVersionAction(version, nothingToCommit && selectedDiffReleases)}
        </div>
        <p className="u-fontSize--small u-lineHeight--normal u-color--dustyGray u-fontWeight--medium u-marginTop--15">Deployed: <span className="u-fontWeight--bold">{version.deployedAt ? dayjs(version.deployedAt).format("MMMM D, YYYY @ h:mm a") : "N/A"}</span></p>
      </div>
    </div>
  )
}
