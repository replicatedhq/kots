import React from "react";
import { Link } from "react-router-dom";
import dayjs from "dayjs";
import classNames from "classnames";
import { getClusterType } from "@src/utilities/utilities";

export default function ActiveDownstreamVersionRow(props) {
  const { watch, match } = props;
  const isGit = watch.cluster.gitOpsRef;
  const icon = getClusterType(isGit) === "git" ? "icon github-small-size" : "icon ship-small-size";
  const { owner, slug } = match.params;
  return (
    <div className="flex flex-auto ActiveDownstreamVersionRow--wrapper">
      <div className="flex-column flex1">
        <div className="flex flex-auto alignItems--center u-fontWeight--bold u-color--tuna">
          <span className={classNames(icon, "flex-auto u-marginRight--5")} />
          <p className="u-fontSize--small u-fontWeight--medium u-color--tuna">
            {watch.cluster.slug}
          </p>
        </div>
        <div className="flex flex-auto alignItems--center u-marginTop--5">
          <span className="u-fontSize--largest u-color--tuna u-lineHeight--normal u-fontWeight--bold u-marginRight--10">{watch.currentVersion ? watch.currentVersion.title : "---"}</span>
          {!watch.currentVersion &&
            <div className="flex-auto flex alignItems--center alignSelf--center">
              <div className="icon blueCircleMinus--icon"></div>
              <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-marginLeft--5">No deployments made</p>
            </div>
          }
          {watch.currentVersion && watch.pendingVersions?.length === 1 &&
            <div className="flex-auto flex alignItems--center alignSelf--center">
              <div className="icon exclamationMark--icon"></div>
              <p className="u-fontSize--normal u-color--orange u-fontWeight--medium u-marginLeft--5">One version behind</p>
            </div>
          }
          {watch.currentVersion && watch.pendingVersions?.length >= 2 &&
            <div className="flex-auto flex alignItems--center alignSelf--center">
              <div className="icon exclamationMark--icon"></div>
              <p className="u-fontSize--normal u-color--orange u-fontWeight--medium u-marginLeft--5">Two or more versions behind</p>
            </div>
          }
          {watch.currentVersion && !watch.pendingVersions?.length &&
            <div className="flex-auto flex alignItems--center alignSelf--center">
              <div className="icon checkmark-icon"></div>
              <p className="u-fontSize--normal u-color--nevada u-fontWeight--medium u-marginLeft--5">Up to date</p>
            </div>
          }
          {watch.currentVersion?.deployedAt && <span className="u-fontSize--small u-color--dustyGray u-fontWeight--medium u-lineHeight--normal u-marginLeft--5">({isGit ? "Merged" : "Deployed"} on {dayjs(watch.currentVersion.deployedAt).format("MMMM D, YYYY")})</span>}
        </div>
      </div>
      <div className="flex-auto flex-column justifyContent--center">
          {isGit && !watch.currentVersion ?
            <a href={`https://github.com/${watch.cluster.gitOpsRef.owner}/${watch.cluster.gitOpsRef.repo}/pull/${watch.pendingVersions[0]?.pullrequestNumber}`} className="btn secondary small" target="_blank" rel="noopener noreferrer">Review PR to deploy application</a>
          :
          <Link to={`/watch/${owner}/${slug}/downstreams/${watch.slug}/version-history`} className="btn secondary small">View downstream {isGit ? "history" : "updates"}</Link>
          }
      </div>
    </div>
  )

}