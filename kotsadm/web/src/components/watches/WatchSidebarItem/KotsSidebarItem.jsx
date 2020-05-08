import React from "react";
import classNames from "classnames";
import { Link } from "react-router-dom";

export default function KotsSidebarItem(props) {
  const { className, app } = props;
  const { iconUri, name, slug } = app;

  let downstreamPendingLengths = [];
  app.downstreams?.map((w) => { 
    if (w.currentVersion) {
      downstreamPendingLengths.push(w.pendingVersions?.length);
    }
  });

  let versionsBehind;
  if (downstreamPendingLengths?.length) {
    versionsBehind = Math.max(...downstreamPendingLengths);
  }

  const isBehind = versionsBehind >= 2
    ? "2+"
    : versionsBehind;

  let versionsBehindText =  "Up to date";
  if (!app.downstreams?.length) {
    versionsBehindText = "No downstreams found"
  } else if (isBehind) {
    versionsBehindText = `${isBehind} ${isBehind >= 2 || typeof isBehind === 'string' ? "versions" : "version"} behind`
  }

  const gitopsEnabled = app.downstreams?.length > 0 && app.downstreams[0].gitops?.enabled;

  return (
    <div className={classNames('sidebar-link', className)}>
      <Link
        className="flex alignItems--center"
        to={`/app/${slug}`}>
          <span className="sidebar-link-icon" style={{ backgroundImage: `url(${iconUri})` }}></span>
          <div className="flex-column">
            <p className={classNames("u-color--tuna u-fontWeight--bold", { "u-marginBottom--10": !gitopsEnabled })}>{name}</p>
            {!gitopsEnabled &&
              <div className="flex alignItems--center">
                <div className={classNames("icon", {
                  "checkmark-icon": !isBehind,
                  "exclamationMark--icon": isBehind,
                  "grayCircleMinus--icon": !app.downstreams?.length
                })}
                />
                <span className={classNames("u-marginLeft--5 u-fontSize--normal u-fontWeight--medium", {
                  "u-color--dustyGray": !isBehind,
                  "u-color--orange": isBehind
                })}>
                  {versionsBehindText}
                </span>
              </div>
            }
          </div>
      </Link>
    </div>
  );
}
