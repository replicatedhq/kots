import React from "react";
import Select from "react-select";
import Modal from "react-modal";
import { Utilities } from "../../utilities/utilities";
import enabled from "../../images/enabled.svg";
import not_enabled from "../../images/not_enabled.svg";
import warning from "../../images/warning.svg";
import styled from "styled-components";
import DisableModal from "./modals/DisableModal";
import { useHistory } from "react-router";

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
}) => {
  const {
    hostname,
    httpPort,
    sshPort,
    services,
    selectedService,
    providerError,
    finishingSetup,
  } = state;
  const apps = appsList.map((app) => ({
    ...app,
    value: app.name,
    label: app.name,
  }));
  const [app, setApp] = React.useState({});

  React.useEffect(() => {
    if (appsList.length > 0) {
      setApp(
        appsList.find((app) => {
          return app.id === selectedApp?.id;
        })
      );
    }
  }, [selectedApp, appsList]);

  const history = useHistory();

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
      console.log("res", res);
      if (!res.ok && res.status === 401) {
        Utilities.logoutUser();
        return;
      }
      if (res.ok && res.status === 204) {
        history.push(`/app/${app?.slug}`);
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
          <div>
            <span style={{ fontSize: 14 }}>{app.label}</span>{" "}
          </div>
          <div>
            {!gitopsEnabled && !gitopsConnected ? (
              <div className="flex" style={{ gap: "5px", color: "gray" }}>
                <img src={not_enabled} alt="not_enabled" />
                <p>Not Enabled</p>
              </div>
            ) : gitopsEnabled && !gitopsConnected ? (
              <div className="flex" style={{ gap: "5px", color: "orange" }}>
                <img src={warning} alt="warning" />
                <p>Enabled, repository access needed</p>
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

  return (
    <div
      key={`${step.step}-active`}
      className="GitOpsDeploy--step u-textAlign--left"
    >
      <p className="step-title">{step.title}</p>
      <p className="step-sub">
        Connect a git version control system so all application updates are
        committed to a git repository. When GitOps is enabled, you cannot deploy
        updates directly from the admin console.
      </p>
      <div className="flex-column u-textAlign--left u-marginBottom--30">
        <div className="flex alignItems--center">
          <div className="flex flex1 flex-column u-marginRight--10">
            <p className="u-fontSize--large u-textColor--primary u-fontWeight--bold u-lineHeight--normal">
              Select an application to configure
            </p>

            <div className="u-position--relative u-marginTop--5 u-marginBottom--40">
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
          <div className="flex flex1 flex-column u-fontSize--small ">
            <a
              style={{ color: "blue", cursor: "pointer" }}
              disabled={disablingGitOps}
              onClick={promptToDisableGitOps}
            >
              {disablingGitOps
                ? "Disabling GitOps"
                : "Disable GitOps for this app"}
            </a>
          </div>
        </div>
        {/* <div className="flex flex1"> */}
        {renderGitOpsProviderSelector({
          provider,
          hostname,
          httpPort,
          sshPort,
          providerError,
          services,
          selectedService,
        })}

        {/* </div> */}
        {/* {isBitbucketServer && (
          <div className="flex flex1 u-marginTop--30">
            {renderHttpPort(provider, httpPort)}
            {renderSshPort(provider, sshPort)}
          </div>
        )} */}
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
