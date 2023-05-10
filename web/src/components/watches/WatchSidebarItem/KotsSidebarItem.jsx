import React from "react";
import classNames from "classnames";
import { Link } from "react-router-dom";
import Icon from "@src/components/Icon";

export default function KotsSidebarItem(props) {
  const { className, app } = props;
  const { slug } = app;
  const name = app.downstream?.currentVersion?.appTitle || app.name;
  const iconUri = app.downstream?.currentVersion?.appIconUri || app.iconUri;

  let versionsBehind;
  if (app.downstream?.currentVersion) {
    versionsBehind = app.downstream?.pendingVersions?.length;
  }

  const isBehind = versionsBehind >= 2 ? "2+" : versionsBehind;

  let versionsBehindText = "Up to date";
  if (!app.downstream) {
    versionsBehindText = "No downstream found";
  } else if (isBehind) {
    versionsBehindText = `${isBehind} ${
      isBehind >= 2 || typeof isBehind === "string" ? "versions" : "version"
    } behind`;
  }

  const gitopsIsConnected = app.downstream?.gitops?.isConnected;

  return (
    <div className={classNames("sidebar-link", className)}>
      <Link className="flex alignItems--center" to={`/app/${slug}`}>
        <span
          className="sidebar-link-icon"
          style={{ backgroundImage: `url(${iconUri})` }}
        ></span>
        <div className="flex-column">
          <p
            className={classNames(
              "u-textColor--primary u-fontWeight--bold break-word",
              {
                "u-marginBottom--10": !gitopsIsConnected,
              }
            )}
          >
            {name}
          </p>
          {!gitopsIsConnected && (
            <div className="flex alignItems--center">
              <Icon
                icon={
                  isBehind
                    ? "warning-circle-filled"
                    : !isBehind
                    ? "check-circle-filled"
                    : !app.downstream && "no-activity-circle-filled--icon"
                }
                className={
                  isBehind
                    ? "warning-color"
                    : !isBehind
                    ? "success-color"
                    : !app.downstream && "no-activity-circle-filled--icon"
                }
                size={16}
              />
              <span
                className={classNames(
                  "u-marginLeft--5 u-fontSize--normal u-fontWeight--medium",
                  {
                    "u-textColor--bodyCopy": !isBehind,
                    "u-textColor--warning": isBehind,
                  }
                )}
              >
                {versionsBehindText}
              </span>
            </div>
          )}
        </div>
      </Link>
    </div>
  );
}
