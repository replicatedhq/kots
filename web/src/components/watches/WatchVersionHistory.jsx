import React, { Fragment } from "react";
import classNames from "classnames";
import truncateMiddle from "truncate-middle";
import { getClusterType } from "@src/utilities/utilities";

import "@src/scss/components/watches/WatchVersionHistory.scss";

export default function WatchVersionHistory(props) {
  const { watch } = props;

  // Sanity check for null watches
  if (!watch) {
    return null;
  }

  const { currentVersion, pendingVersions, watches, pastVersions } = watch;
  const versionHistory = pendingVersions.concat(currentVersion, pastVersions);

  let clustersNode;
  if (watches?.length > 0) {
    clustersNode = (
      <Fragment>
        {watches.map(({ cluster }) => {
          const icon = getClusterType(cluster.gitOpsRef) === "git"
            ? "icon github-small-size"
            : "icon ship-small-size";

          return (
            <div key={cluster.slug} className="watch-cell flex">
              <div className="flex flex1 cluster-cell-title justifyContent--center alignItems--center u-fontWeight--bold u-color--tuna">
                <span className={classNames(icon, "flex-auto u-marginRight--5")} />
                <p className="u-fontSize--small u-fontWeight--medium u-color--tuna">
                  {truncateMiddle(cluster.slug, 8, 6, "...")}
                </p>
              </div>
            </div>
          );
        })}
      </Fragment>
    );
  } else if (watch.cluster) {
    const icon = getClusterType(watch.cluster.gitOpsRef) === "git"
      ? "icon github-small-size"
      : "icon ship-small-size";
    clustersNode = (
      <Fragment>
        <div key={watch.cluster.slug} className="watch-cell flex">
          <div className="flex flex1 cluster-cell-title justifyContent--center alignItems--center u-fontWeight--bold u-color--tuna">
            <span className={classNames(icon, "flex-auto u-marginRight--5")} />
            <p className="u-fontSize--small u-fontWeight--medium u-color--tuna">
              {truncateMiddle(watch.cluster.slug, 8, 6, "...")}
            </p>
          </div>
        </div>
      </Fragment>
    );
  }

  return (
    <div className="centered-container flex-column u-position--relative u-overflow--auto">
      <div className="flex alignItems--center u-borderBottom--gray u-paddingBottom--5">
        <p className="u-fontSize--header u-fontWeight--bold u-color--tuna">
          {currentVersion ? currentVersion.title : "Unknown"}
        </p>
        <div className={classNames("icon flex-auto u-marginLeft--10 u-marginRight--5",{
            "checkmark-icon": currentVersion,
            "blueCircleMinus--icon": !currentVersion
          })}/>
        <p className="u-fontSize--large">{currentVersion ? "Most recent version" : "No deployments made"}</p>
        {!watch.cluster && <p className="u-fonSize--small u-marginLeft--10 replicated-link"><button className="btn secondary small" onClick={props.onCheckForUpdates}>Check for update</button></p>}
        <div className="flex flex1 justifyContent--flexEnd">
          {clustersNode}
        </div>
      </div>
      <div className="flex-column">
        {versionHistory.length > 0 && versionHistory.map( version => {
          console.log(version);
          if (!version) return null;
          return (
            <div
              key={`${version.title}-${version.sequence}`}
              className="flex u-paddingTop--20 u-paddingBottom--20 u-borderBottom--gray">
              <div className="flex alignItems--center u-fontSize--larger u-color--tuna u-fontWeight--bold u-marginLeft--10">
                Version {version.title}
                {version.pullrequestNumber &&
                  <div>
                    <span className="icon integration-card-icon-github u-marginRight--5 u-marginLeft--5" />
                    <a
                      className="u-color--astral u-marginLeft--5"
                      href=""
                      target="_blank"
                      rel="norefeer nofollow">
                        #{version.pullrequestNumber}
                    </a>
                  </div>
                }
              </div>
              <div className="flex flex1 justifyContent--flexEnd alignItems--center">
                <div className="watch-cell">
                  <div className="flex justifyContent--center alignItems--center">
                    <div className={classNames("icon", {
                      "checkmark-icon": version.status === "deployed",
                      "exclamationMark-icon": version.status !== "deployed"
                    })}
                    />
                  </div>
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
