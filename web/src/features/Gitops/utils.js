import enabled from "../../images/enabled.svg";
import not_enabled from "../../images/not_enabled.svg";
import warning from "../../images/warning.svg";
import { IconWrapper } from "./constants";

const renderIcons = (app) => {
  const appIconUri =
    app?.downstream?.currentVersion?.appIconUri || app?.iconUri;
  if (appIconUri) {
    return (
      <IconWrapper
        style={{ backgroundImage: `url(${appIconUri})` }}
      ></IconWrapper>
    );
  }
};
export const getLabel = (app, isSingleApp) => {
  const downstream = app?.downstream;
  const gitops = downstream?.gitops;
  const gitopsEnabled = gitops?.enabled;
  const gitopsConnected = gitops?.isConnected;
  const appLabel = app?.downstream?.currentVersion?.appTitle || app?.label;

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
              {appLabel}
            </span>
          ) : (
            <span style={{ fontSize: 14 }}>{appLabel}</span>
          )}
        </div>
        <div style={{ fontSize: "14px" }}>
          {!gitopsEnabled && !gitopsConnected ? (
            <div className="flex gray-color" style={{ gap: "5px" }}>
              <img src={not_enabled} alt="not_enabled" />
              <p data-testid="gitops-not-enabled">Not Enabled</p>
            </div>
          ) : gitopsEnabled && !gitopsConnected ? (
            <div className="flex warning-color" style={{ gap: "5px" }}>
              <img src={warning} alt="warning" />
              <p data-testid="gitops-repository-access-needed">
                Repository access needed
              </p>
            </div>
          ) : (
            gitopsEnabled &&
            gitopsConnected && (
              <div className="flex success-color" style={{ gap: "5px" }}>
                <img src={enabled} alt="enabled" />
                <p data-testid="gitops-enabled">Enabled</p>
              </div>
            )
          )}
        </div>
      </div>
    </div>
  );
};
