import { useContext } from "react";
import { Flex, Paragraph } from "../../../styles/common";
import Select from "react-select";
import { GitOpsContext, withGitOpsConsumer } from "../context";
import {
  BITBUCKET_SERVER_DEFAULT_SSH_PORT,
  BITBUCKET_SERVER_DEFAULT_HTTP_PORT,
} from "../constants";

export function requiresHostname(provider) {
  return (
    provider === "gitlab_enterprise" ||
    provider === "github_enterprise" ||
    provider === "bitbucket_server" ||
    provider === "other"
  );
}

const GitOpsProviderSelector = () => {
  const {
    services,
    selectedService,
    handleServiceChange,
    sshPort,
    setSshPort,
    httpPort,
    setHttpPort,
    hostname,
    setHostname,
    providerError,
    provider,
  } = useContext(GitOpsContext);

  const isBitbucketServer = provider === "bitbucket_server";
  const renderIcons = (service) => {
    if (service) {
      return <span className={`icon gitopsService--${service.value}`} />;
    } else {
      return;
    }
  };

  const getLabel = (service, label) => {
    return (
      <div style={{ alignItems: "center", display: "flex" }}>
        <span style={{ fontSize: 18, marginRight: "10px" }}>
          {renderIcons(service)}
        </span>
        <span style={{ fontSize: 14 }}>{label}</span>
      </div>
    );
  };

  const renderHttpPort = (httpPort) => {
    if (isBitbucketServer) {
      return (
        <Flex flex="1" direction="column" width="100%">
          <p className="card-item-title">
            HTTP Port <span className="card-item-title">(Required)</span>
          </p>
          <input
            type="text"
            className="Input"
            placeholder={BITBUCKET_SERVER_DEFAULT_HTTP_PORT}
            value={httpPort}
            onChange={(e) => setHttpPort(e.target.value)}
          />
        </Flex>
      );
    }
  };

  const renderSshPort = (sshPort) => {
    if (!isBitbucketServer) {
      return <div className="flex flex1" />;
    }
    return (
      <div className="flex flex1 flex-column">
        <p className="card-item-title">
          SSH Port <span className="card-item-title">(Required)</span>
        </p>
        <input
          type="text"
          className="Input"
          placeholder={BITBUCKET_SERVER_DEFAULT_SSH_PORT}
          value={sshPort}
          onChange={(e) => setSshPort(e.target.value)}
        />
      </div>
    );
  };

  const renderHostName = (provider, hostname, providerError) => {
    if (requiresHostname(provider)) {
      return (
        <Flex direction="column" className="flex1" width="100%">
          <p className="card-item-title">
            Hostname
            <span className="card-item-title"> (Required)</span>
          </p>
          <input
            type="text"
            className={`Input ${
              providerError?.field === "hostname" && "has-error"
            } u-marginTop--5`}
            placeholder="hostname"
            value={hostname}
            onChange={(e) => setHostname(e.target.value)}
          />
          {providerError?.field === "hostname" && (
            <p className="u-fontSize--small u-marginTop--5 u-textColor--error u-fontWeight--medium u-lineHeight--normal">
              A hostname must be provided
            </p>
          )}
        </Flex>
      );
    }
  };

  return (
    <Flex direction="column">
      <Flex width="100%">
        {/* left column */}
        <Flex direction="column" flex="1" mr="20">
          <div style={{ width: "100%" }}>
            <p className="card-item-title">Git provider</p>
            <div className="u-position--relative  u-marginTop--5">
              <Select
                className="replicated-select-container"
                classNamePrefix="replicated-select"
                placeholder="Select a GitOps service"
                options={services}
                isSearchable={false}
                getOptionLabel={(service) => getLabel(service, service.label)}
                getOptionValue={(service) => service.label}
                value={selectedService}
                onChange={handleServiceChange}
                isOptionSelected={(option) => {
                  option.value === selectedService;
                }}
              />
            </div>
          </div>

          {isBitbucketServer && (
            <Flex flex="1" mt="30" width="100%">
              {renderHttpPort(httpPort)}
            </Flex>
          )}
        </Flex>
        <Flex direction="column" flex="1" width="100%">
          {/* right column */}
          {renderHostName(provider, hostname, providerError, httpPort, sshPort)}
          {isBitbucketServer && (
            <Flex flex="1" mt="30" width="100%">
              {renderSshPort(sshPort)}
            </Flex>
          )}
        </Flex>
      </Flex>
    </Flex>
  );
};

export default withGitOpsConsumer(GitOpsProviderSelector);
