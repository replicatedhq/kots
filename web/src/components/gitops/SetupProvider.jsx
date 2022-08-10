import React from "react";
import Select from "react-select";
import { requiresHostname } from "../../utilities/utilities";

const BITBUCKET_SERVER_DEFAULT_HTTP_PORT = "7990";
const BITBUCKET_SERVER_DEFAULT_SSH_PORT = "7999";

const SetupProvider = ({
  step,
  state,
  provider,
  handleServiceChange,
  updateHostname,
  updateHttpPort,
  updateSSHPort,
  updateSettings,
  isSingleApp,
  getLabel,
  renderGitOpsProviderSelector,
  renderHostName,
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
  const isBitbucketServer = provider === "bitbucket_server";

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
        Before the Admin Console can push changes to your Git repository, some
        information about your Git configuration is required.
      </p>
      <div className="flex-column u-textAlign--left u-marginBottom--30">
        <div className="flex flex1">
          {renderGitOpsProviderSelector(services, selectedService)}
          {renderHostName(provider, hostname, providerError)}
        </div>
        {isBitbucketServer && (
          <div className="flex flex1 u-marginTop--30">
            {renderHttpPort(provider, httpPort)}
            {renderSshPort(provider, sshPort)}
          </div>
        )}
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
