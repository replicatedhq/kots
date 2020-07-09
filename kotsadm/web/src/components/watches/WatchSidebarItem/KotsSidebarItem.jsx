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
          {props.sidebarOpen &&
            <div className="flex-column u-marginLeft--10">
              <p className={classNames("u-color--tuna u-fontSize--normal u-fontWeight--bold", { "u-marginBottom--5": !gitopsEnabled })}>{name}</p>
              {!gitopsEnabled &&
                <div className="flex alignItems--center">
                  <span className={classNames("u-fontSize--small u-fontWeight--medium", {
                    "u-color--chateauGreen": !isBehind,
                    "u-color--orange": isBehind,
                    "u-color--dustyGray": !app.downstreams?.length
                  })}>
                    {versionsBehindText}
                  </span>
                </div>
              }
            </div>
          }
      </Link>
    </div>
  );
}
