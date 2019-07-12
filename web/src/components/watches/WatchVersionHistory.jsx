import React, { Fragment } from "react";
import Helmet from "react-helmet";
import classNames from "classnames";
import truncateMiddle from "truncate-middle";
import Loader from "../shared/Loader";
import { getClusterType } from "@src/utilities/utilities";

import "@src/scss/components/watches/WatchVersionHistory.scss";

export default function WatchVersionHistory(props) {
  const { watch, checkingForUpdates, checkingUpdateText, errorCheckingUpdate } = props;

  // Sanity check for null watches
  if (!watch) {
    return null;
  }

  const { currentVersion, pendingVersions, watches, pastVersions } = watch;
  const versionHistory = pendingVersions.concat(currentVersion, pastVersions);

  let clustersNode;
  let checkUpdateNode;
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

  if (checkingForUpdates) {
    checkUpdateNode = (
      <div className="flex alignItems--center">
        <Loader size="26" />
        <span className="u-marginLeft--5 u-fontSize--small u-color--nevada u-fontWeight--medium">{checkingUpdateText}</span>
      </div>
    );
  } else if (errorCheckingUpdate) {
    checkUpdateNode = <p className="u-marginLeft--5 u-fontSize--small u-color--chestnut u-fontWeight--medium">Error checking for updates <span onClick={props.onCheckForUpdates} className="u-fontWeight--bold u-textDecoration--underline u-cursor--pointer">Try again</span></p>
  } else {
    checkUpdateNode = <button className="btn secondary small" onClick={props.onCheckForUpdates}>Check for update</button>
  }

  return (
    <div className="flex-column u-position--relative u-overflow--auto u-padding--20">
      <Helmet>
        <title>{`${watch.watchName} Version History`}</title>
      </Helmet>
      <div className="flex alignItems--center u-borderBottom--gray u-paddingBottom--5">
        <p className="u-fontSize--header u-fontWeight--bold u-color--tuna">
          {currentVersion ? currentVersion.title : "Unknown"}
        </p>
        <div className={classNames("icon flex-auto u-marginLeft--10 u-marginRight--5",{
            "checkmark-icon": currentVersion,
            "blueCircleMinus--icon": !currentVersion
          })}/>
        <p className="u-fontSize--large">{currentVersion ? "Most recent version" : "No deployments made"}</p>
        {!watch.cluster && <div className="u-marginLeft--10">{checkUpdateNode}</div>}
        <div className="flex flex1 justifyContent--flexEnd">
          {clustersNode}
        </div>
      </div>
      <div className="flex-column">
        {versionHistory.length > 0 && versionHistory.map( version => {
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
                      rel="noopener noreferrer">
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
