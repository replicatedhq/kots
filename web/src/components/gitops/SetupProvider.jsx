import React from "react";
import Select from "react-select";
import Modal from "react-modal";
import { Utilities } from "../../utilities/utilities";
import enabled from "../../images/enabled.svg";
import not_enabled from "../../images/not_enabled.svg";
import warning from "../../images/warning.svg";
import styled from "styled-components";
import DisableModal from "./modals/DisableModal";

const BITBUCKET_SERVER_DEFAULT_HTTP_PORT = "7990";
const BITBUCKET_SERVER_DEFAULT_SSH_PORT = "7999";

const IconWrapper = styled.div`
  height: 30px;
  width: 30px;
  border-radius: 50%;
  background-position: center;
  background-size: contain;
  background-repeat: no-repeat;
  box-shadow: inset 0 0 3px rgba(0, 0, 0, 0.3);
  background-color: #ffffff;
  z-index: 1;
`;

const SetupProvider = ({
  step,
  appsList,
  state,
  provider,
  updateHttpPort,
  updateSSHPort,
  updateSettings,
  isSingleApp,
  renderGitOpsProviderSelector,
  renderHostName,
  handleAppChange,
  selectedApp,
  finishSetup,
  getAppsList,
  getGitops,
}) => {
  const {
    owner,
    repo,
    branch,
    path,
    hostname,
    httpPort,
    sshPort,
    services,
    selectedService,
    providerError,
    finishingSetup,
  } = state;

  const [app, setApp] = React.useState({});
  const apps = appsList?.map((app) => ({
    ...app,
    value: app.name,
    label: app.name,
  }));

  React.useEffect(() => {
    //TO DO: will refactor in next PR
    const apps = appsList?.map((app) => ({
      ...app,
      value: app.name,
      label: app.name,
    }));
    if (appsList.length > 0) {
      setApp(
        apps.find((app) => {
          return app.id === selectedApp?.id;
        })
      );
    }
  }, [selectedApp, appsList]);

  const [showDisableGitopsModalPrompt, setShowDisableGitopsModalPrompt] =
    React.useState(false);
  const [disablingGitOps, setDisablingGitOps] = React.useState(false);

  const promptToDisableGitOps = () => {
    setShowDisableGitopsModalPrompt(true);
  };

  const disableGitOps = async () => {
    setDisablingGitOps(true);

    const appId = app?.id;
    let clusterId;
    if (app?.downstream) {
      clusterId = app.downstream.cluster.id;
    }

    try {
      const res = await fetch(
        `${process.env.API_ENDPOINT}/gitops/app/${appId}/cluster/${clusterId}/disable`,
        {
          headers: {
            Authorization: Utilities.getToken(),
            "Content-Type": "application/json",
          },
          method: "POST",
        }
      );
      if (!res.ok && res.status === 401) {
        Utilities.logoutUser();
        return;
      }
      if (res.ok && res.status === 204) {
        getAppsList();
        getGitops();

        setShowDisableGitopsModalPrompt(false);
      }
    } catch (err) {
      console.log(err);
    } finally {
      setDisablingGitOps(false);
    }
  };
  const renderIcons = (app) => {
    if (app?.iconUri) {
      return (
        <IconWrapper
          style={{ backgroundImage: `url(${app?.iconUri})` }}
        ></IconWrapper>
      );
    }
  };
  const getLabel = (app) => {
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

  const downstream = app?.downstream;
  const gitops = downstream?.gitops;
  const gitopsEnabled = gitops?.enabled;
  const gitopsConnected = gitops?.isConnected;

  return (
    <div
      key={`${step.step}-active`}
      className="GitOpsDeploy--step u-textAlign--left"
    >
      <p className="step-title">{step.title}</p>
      <p className="step-sub">
        Connect a git version control system so all application updates are
        committed to a git <br />
        repository. When GitOps is enabled, you cannot deploy updates directly
        from the <br />
        admin console.
      </p>
      <div className="flex-column u-textAlign--left ">
        <div className="flex alignItems--center u-marginBottom--30">
          {isSingleApp && app ? (
            <div className="u-marginRight--5">{getLabel(app)}</div>
          ) : (
            <div className="flex flex1 flex-column u-marginRight--10">
              <p className="u-fontSize--large u-textColor--primary u-fontWeight--bold u-lineHeight--normal">
                Select an application to configure
              </p>
              <div className="u-position--relative u-marginTop--5 u-marginBottom--10">
                <Select
                  className="replicated-select-container select-large "
                  classNamePrefix="replicated-select"
                  placeholder="Select an application"
                  options={apps}
                  isSearchable={false}
                  getOptionLabel={(app) => getLabel(app)}
                  // getOptionValue={(app) => app.label}
                  value={selectedApp}
                  onChange={handleAppChange}
                  isOptionSelected={(option) => {
                    option.value === selectedApp;
                  }}
                />
              </div>
            </div>
          )}
          <div className="flex flex1 flex-column u-fontSize--small u-marginTop--20">
            {gitopsEnabled && gitopsConnected && (
              <a
                style={{ color: "blue", cursor: "pointer" }}
                disabled={disablingGitOps}
                onClick={promptToDisableGitOps}
              >
                {disablingGitOps
                  ? "Disabling GitOps"
                  : "Disable GitOps for this app"}
              </a>
            )}
          </div>
        </div>
        {renderGitOpsProviderSelector({
          owner,
          repo,
          branch,
          path,
          provider,
          hostname,
          httpPort,
          sshPort,
          providerError,
          services,
          selectedService,
        })}
      </div>
      <div>
        <DisableModal
          isOpen={showDisableGitopsModalPrompt}
          setOpen={setShowDisableGitopsModalPrompt}
          disableGitOps={disableGitOps}
          provider={provider}
        />
      </div>
    </div>
  );
};

export default SetupProvider;
