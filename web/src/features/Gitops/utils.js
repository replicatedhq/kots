import React from "react";
import enabled from "../../images/enabled.svg";
import not_enabled from "../../images/not_enabled.svg";
import warning from "../../images/warning.svg";
import { IconWrapper } from "./constants";

const renderIcons = (app) => {
  if (app?.iconUri) {
    return (
      <IconWrapper
        style={{ backgroundImage: `url(${app?.iconUri})` }}
      ></IconWrapper>
    );
  }
};
export const getLabel = (app, isSingleApp) => {
  const downstream = app?.downstream;
  const gitops = downstream?.gitops;
  const gitopsEnabled = gitops?.enabled;
  const gitopsConnected = gitops?.isConnected;

  return (
    <div style={{ alignItems: "center", display: "flex" }}>
      <span style={{ fontSize: 18, marginRight: "10px" }}>
        {renderIcons(app)}
      </span>
      <div className="flex flex-column">
        <div className={isSingleApp && "u-marginBottom--5"}>
          {isSingleApp ? (
            <span
              style={{
                fontSize: "16",
                fontWeight: "bold",
                color: "#323232",
              }}
            >
              {app.label}
            </span>
          ) : (
            <span style={{ fontSize: 14 }}>{app.label}</span>
          )}
        </div>
        <div style={{ fontSize: "14px" }}>
          {!gitopsEnabled && !gitopsConnected ? (
            <div className="flex" style={{ gap: "5px", color: "gray" }}>
              <img src={not_enabled} alt="not_enabled" />
              <p>Not Enabled</p>
            </div>
          ) : gitopsEnabled && !gitopsConnected ? (
            <div className="flex" style={{ gap: "5px", color: "orange" }}>
              <img src={warning} alt="warning" />
              <p>Repository access needed</p>
            </div>
          ) : (
            gitopsEnabled &&
            gitopsConnected && (
              <div className="flex" style={{ gap: "5px", color: "green" }}>
                <img src={enabled} alt="enabled" />
                <p>Enabled</p>
              </div>
            )
          )}
        </div>
      </div>
    </div>
  );
};
