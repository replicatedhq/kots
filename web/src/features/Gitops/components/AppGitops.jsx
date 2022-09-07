import React, { useContext, useState, useEffect } from "react";
import Helmet from "react-helmet";
import CodeSnippet from "@src/components/shared/CodeSnippet";
import { getAddKeyUri, Utilities } from "../../../utilities/utilities";
import { useHistory } from "react-router-dom";
import ConnectionModal from "./modals/ConnectionModal";
import Loader from "../../../components/shared/Loader";
import DisableModal from "./modals/DisableModal";
import { Flex } from "../../../styles/common";
import { withGitOpsConsumer, GitOpsContext } from "../context";
import { getLabel } from "../utils";
import { SERVICES } from "../constants";
import AppSelector from "./AppSelector";

import "../../../scss/components/gitops/GitOpsDeploymentManager.scss";
import "../../../scss/components/gitops/GitOpsSettings.scss";
import "../../../scss/components/gitops/GitopsPrism.scss";

const AppGitops = () => {
  const [ownerRepo, setOwnerRepo] = useState("");
  const [testingConnection, setTestingConnection] = useState(false);
  const [disablingGitOps, setDisablingGitOps] = useState(false);
  const [showDisableGitopsModalPrompt, setShowDisableGitopsModalPrompt] =
    useState(false);
  const [showConnectionModal, setShowConnectionModal] = useState(false);
  const [modalType, setModalType] = useState("");

  const {
    selectedApp,
    appsList,
    handleAppChange,
    stepFrom,
    isSingleApp,
    gitopsConnected,
    gitopsEnabled,
    getAppsList,
  } = useContext(GitOpsContext);

  const history = useHistory();

  useEffect(() => {
    getInitialOwnerRepo();

    if (!gitopsEnabled) {
      history.push(`/app/${selectedApp.slug}`);
    }
  }, []);

  const gitops = selectedApp?.downstream.gitops;
  const deployKey = gitops?.deployKey;
  const addKeyUri = getAddKeyUri(gitops, ownerRepo);

  const selectedService = SERVICES.find((service) => {
    return service.value === gitops?.provider;
  });

  const getInitialOwnerRepo = () => {
    if (!selectedApp?.downstream) {
      return "";
    }

    const gitops = selectedApp.downstream.gitops;
    if (!gitops?.uri) {
      return "";
    }

    let ownerRepo = "";
    const parsed = new URL(gitops?.uri);
    if (gitops?.provider === "bitbucket_server") {
      const project =
        parsed.pathname.split("/").length > 2 && parsed.pathname.split("/")[2];
      const repo =
        parsed.pathname.split("/").length > 4 && parsed.pathname.split("/")[4];
      if (project && repo) {
        ownerRepo = `${project}/${repo}`;
      }
    } else {
      ownerRepo = parsed.pathname.slice(1); // remove the "/"
    }
    setOwnerRepo(ownerRepo);
  };

  const handleTestConnection = async () => {
    setTestingConnection(true);

    const appId = selectedApp?.id;
    let clusterId;
    if (selectedApp?.downstream) {
      clusterId = selectedApp.downstream.cluster.id;
    }
    // TODO: update this to react query hook
    try {
      const res = await fetch(
        `${process.env.API_ENDPOINT}/gitops/app/${appId}/cluster/${clusterId}/initconnection`,
        {
          method: "POST",
          headers: {
            Authorization: Utilities.getToken(),
            "Content-Type": "application/json",
          },
        }
      );
      if (!res.ok) {
        if (res.status === 401) {
          Utilities.logoutUser();
          return;
        }

        if (res.status === 400) {
          const response = await res.json();
          if (response?.error) {
            setShowConnectionModal(true);
            setModalType("fail");
            console.log(response?.error);
          }
          throw new Error(`authentication failed`);
        }
        throw new Error(`unexpected status code: ${res.status}`);
      }

      setShowConnectionModal(true);
      setModalType("success");
    } catch (err) {
      console.log(err);
      setModalType("fail");
    } finally {
      setTestingConnection(false);
    }
  };

  const promptToDisableGitOps = () => {
    setShowDisableGitopsModalPrompt(true);
  };

  const disableGitOps = async () => {
    setDisablingGitOps(true);

    const appId = selectedApp?.id;
    let clusterId;
    if (selectedApp?.downstream) {
      clusterId = selectedApp.downstream.cluster.id;
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
        await getAppsList();
      }
    } catch (err) {
      console.log(err);
    } finally {
      setDisablingGitOps(false);
      setShowDisableGitopsModalPrompt(false);
    }
  };

  // something funky is happening here, if we use appsList from context,
  // the app selector hover state gets messed up
  // this is just a temp fix
  const apps = appsList?.map((app) => ({
    ...app,
    value: selectedApp.name,
    label: selectedApp.name,
  }));

  const appTitle = selectedApp?.name;
  return (
    <div className="GitOpsDeploy--step u-textAlign--left">
      <Helmet>
        <title>{`${appTitle} GitOps`}</title>
      </Helmet>
      <div className="flex-column flex1">
        <div className="GitopsSettings-noRepoAccess u-textAlign--left">
          <p className="step-title">GitOps Configuration</p>
          <p className="step-sub">
            Connect a git version control system so all application updates are
            committed to a git <br />
            repository. When GitOps is enabled, you cannot deploy updates
            directly from the <br />
            admin console.
          </p>
        </div>
        <div className="flex alignItems--center u-marginBottom--30">
          {isSingleApp && selectedApp ? (
            <div className="u-marginRight--5">{getLabel(selectedApp)}</div>
          ) : (
            <AppSelector
              apps={apps}
              selectedApp={selectedApp}
              handleAppChange={handleAppChange}
              isSingleApp={isSingleApp}
            />
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

        <div
          style={{
            marginBottom: "30px",
          }}
        >
          <Flex mb="15" align="center">
            <span
              className="icon small-warning-icon"
              style={{ width: "35px" }}
            />
            <p
              className="u-fontSize--large u-fontWeight--bold u-lineHeight--normal"
              style={{ color: "#DB9016" }}
            >
              Access to your repository is needed to push application updates
            </p>
          </Flex>
          <p
            className="u-fontSize--normal u-fontWeight--normal u-marginBottom--15"
            style={{ color: "#585858" }}
          >
            Add this SSH key on your
            <a
              className="replicated-link"
              href={addKeyUri}
              target="_blank"
              rel="noopener noreferrer"
            >
              {selectedApp.downstream.gitops.provider === "bitbucket_server"
                ? " account settings page, "
                : " repository settings page, "}
            </a>
            and grant it write access.
          </p>
          <CodeSnippet
            canCopy={true}
            copyText="Copy key"
            onCopyText={<span className="u-textColor--success">Copied</span>}
          >
            {deployKey}
          </CodeSnippet>
        </div>

        <div className="flex justifyContent--spaceBetween alignItems--center">
          <div className="flex">
            <button
              className="btn secondary blue"
              onClick={() => stepFrom("action", "provider")}
            >
              Back to configuration
            </button>
          </div>
          {testingConnection ? (
            <Loader size="30" />
          ) : (
            <button
              className="btn primary blue"
              disabled={testingConnection}
              onClick={handleTestConnection}
            >
              Test connection to repository
            </button>
          )}
        </div>
      </div>

      <DisableModal
        isOpen={showDisableGitopsModalPrompt}
        setOpen={(e) => setShowDisableGitopsModalPrompt(e)}
        disableGitOps={disableGitOps}
        provider={selectedService}
      />

      <ConnectionModal
        isOpen={showConnectionModal}
        modalType={modalType}
        setOpen={(e) => setShowConnectionModal(e)}
        handleTestConnection={handleTestConnection}
        isTestingConnection={testingConnection}
        stepFrom={stepFrom}
        appSlug={selectedApp.slug}
        getAppsList={getAppsList}
      />
    </div>
  );
};

export default withGitOpsConsumer(AppGitops);
