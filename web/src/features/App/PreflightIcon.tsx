import React from "react";
import Icon from "@components/Icon";
import { Link } from "react-router-dom";
import { App, Version } from "@types";

type Props = {
  app: App | null;
  isNewPreflightResults: boolean;
  showDeployLogs?: boolean;
  showActions?: true | Object | undefined;
  preflightState: {
    preflightsFailed: boolean;
    preflightState: string;
  } | null;
  showText: boolean;
  version: Version;
  className: string;
};

const PreflightIcon = ({
  app,
  version,
  isNewPreflightResults,
  showDeployLogs,
  showActions,
  preflightState,
  showText,
  className,
}: Props) => {
  let checksStatusText;
  let textColor = "";
  if (preflightState?.preflightsFailed) {
    checksStatusText = "Checks failed";
    textColor = "err";
  } else if (preflightState?.preflightState === "warn") {
    checksStatusText = "Checks passed with warnings";
    textColor = "warning";
  } else if (preflightState?.preflightState === "pass") {
    checksStatusText = "Checks passed";
    textColor = "success";
  }

  return (
    <div className="tw-relative">
      <Link
        to={`/app/${app?.slug}/downstreams/${app?.downstream?.cluster?.slug}/version-history/preflight/${version?.sequence}`}
        className={`tw-relative ${className}`}
        data-tip="View preflight checks"
      >
        <Icon icon="preflight-checks" size={22} className="clickable" />
        <>
          {preflightState?.preflightsFailed ? (
            <Icon
              icon={"warning-circle-filled"}
              size={12}
              className="version-row-preflight-status-icon error-color"
            />
          ) : preflightState?.preflightState === "warn" ? (
            <Icon
              icon={"warning"}
              size={12}
              className="version-row-preflight-status-icon warning-color"
            />
          ) : (
            ""
          )}
          {showText && (
            <p
              className={`checks-running-text u-fontSize--small u-lineHeight--normal u-fontWeight--medium ${textColor}
                ${!showDeployLogs && !showActions ? "without-btns" : ""}`}
            >
              {checksStatusText}
            </p>
          )}
        </>
      </Link>
    </div>
  );
};

export default PreflightIcon;
