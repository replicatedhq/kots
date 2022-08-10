import React from "react";
import Select from "react-select";

const BITBUCKET_SERVER_DEFAULT_HTTP_PORT = "7990";
const BITBUCKET_SERVER_DEFAULT_SSH_PORT = "7999";

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

  const apps = appsList.map((app) => ({ value: app.name, label: app.name }));

  const renderHttpPort = (provider, httpPort) => {
    const isBitbucketServer = provider === "bitbucket_server";
    if (!isBitbucketServer) {
      return <div className="flex flex1" />;
    }
    return (
      <div className="flex flex1 flex-column u-marginRight--10">
        <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--normal">
          HTTP Port
        </p>
        <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-lineHeight--normal u-marginBottom--10">
          HTTP Port of your GitOps server.
        </p>
        <input
          type="text"
          className="Input"
          placeholder={BITBUCKET_SERVER_DEFAULT_HTTP_PORT}
          value={httpPort}
          onChange={(e) => updateHttpPort(e.target.value)}
        />
      </div>
    );
  };

  const renderSshPort = (provider, sshPort) => {
    const isBitbucketServer = provider === "bitbucket_server";
    if (!isBitbucketServer) {
      return <div className="flex flex1" />;
    }
    return (
      <div className="flex flex1 flex-column u-marginLeft--10">
        <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--normal">
          SSH Port
        </p>
        <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-lineHeight--normal u-marginBottom--10">
          SSH Port of your GitOps server.
        </p>
        <input
          type="text"
          className="Input"
          placeholder={BITBUCKET_SERVER_DEFAULT_SSH_PORT}
          value={sshPort}
          onChange={(e) => updateSSHPort(e.target.value)}
        />
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
        <div className="flex flex1 flex-column u-marginRight--10">
          <p className="u-fontSize--large u-textColor--primary u-fontWeight--bold u-lineHeight--normal">
            Select an application to configure
          </p>
          <div className="u-position--relative u-marginTop--5 u-marginBottom--40">
            <Select
              className="replicated-select-container"
              classNamePrefix="replicated-select"
              placeholder="Select an application"
              options={apps}
              isSearchable={false}
              // getOptionValue={(service) => service.label}
              value={selectedApp}
              onChange={handleAppChange}
              isOptionSelected={(option) => {
                option.value === selectedApp;
              }}
            />
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
        <button
          className="btn primary blue"
          type="button"
          disabled={finishingSetup}
          onClick={updateSettings}
        >
          {finishingSetup
            ? "Finishing setup"
            : isSingleApp()
            ? "Continue to deployment action"
            : "Finish GitOps setup"}
        </button>
      </div>
    </div>
  );
};

export default SetupProvider;
