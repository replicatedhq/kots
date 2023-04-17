import React, { useState, useEffect } from "react";
import { isEmpty, find } from "lodash";
import {
  getGitOpsUri,
  Utilities,
  requiresHostname,
} from "../../utilities/utilities";
import {
  SERVICES,
  BITBUCKET_SERVER_DEFAULT_HTTP_PORT,
  BITBUCKET_SERVER_DEFAULT_SSH_PORT,
} from "./constants";
import useGitops from "./hooks/useGitops";
const GitOpsContext = React.createContext();

const GitOpsProvider = ({ children }) => {
  const [step, setStep] = useState("provider");
  const [hostname, setHostname] = useState("");
  const [httpPort, setHttpPort] = useState("");
  const [sshPort, setSshPort] = useState("");
  const [services, setServices] = useState(SERVICES);
  const [selectedService, setSelectedService] = useState(SERVICES[0]);
  const [providerError, setProviderError] = useState(null);
  const [finishingSetup, setFinishingSetup] = useState(false);
  const [appsList, setAppsList] = useState([]);
  const [gitops, setGitops] = useState({});
  const [errorMsg, setErrorMsg] = useState("");
  const [errorTitle, setErrorTitle] = useState("");
  const [displayErrorModal, setDisplayErrorModal] = useState(false);
  const [selectedApp, setSelectedApp] = useState({});
  const [owner, setOwner] = useState("");
  const [repo, setRepo] = useState("");
  const [branch, setBranch] = useState("");
  const [path, setPath] = useState("");
  const [gitopsConnected, setGitopsConnected] = useState(false);
  const [gitopsEnabled, setGitopsEnabled] = useState(false);

  const provider = selectedService?.value;

  const { data: freshGitops, refetch: fetchGitops } = useGitops();

  const getInitialOwnerRepo = (app) => {
    // fills out owner,repo... fields
    if (!app?.downstream) {
      setOwner("");
      setRepo("");
      setBranch("");
      setPath("");
      setGitopsConnected(false);
      setGitopsEnabled(false);
      return "";
    }

    const currentGitops = app.downstream.gitops;
    if (!currentGitops?.uri) {
      setOwner("");
      setRepo("");
      setBranch("");
      setPath("");
      setGitopsConnected(currentGitops.enabled);
      setGitopsEnabled(currentGitops.isConnected);
      return "";
    }

    const parsed = new URL(currentGitops?.uri);
    if (currentGitops?.provider === "bitbucket_server") {
      const tempProject =
        parsed.pathname.split("/").length > 2 && parsed.pathname.split("/")[2];
      const tempRepo =
        parsed.pathname.split("/").length > 4 && parsed.pathname.split("/")[4];
      if (tempProject && tempRepo) {
        setOwner(tempProject);
        setRepo(tempRepo);
      }
    } else {
      let tempPath = parsed.pathname.slice(1); // remove the "/"
      const tempProject = tempPath.split("/")[0];
      const tempRepo = tempPath.split("/")[1];
      setOwner(tempProject);
      setRepo(tempRepo);
      setBranch(currentGitops.branch);
      setPath(currentGitops.path);
      setGitopsConnected(currentGitops.enabled);
      setGitopsEnabled(currentGitops.isConnected);
    }
  };

  const getAppsList = async () => {
    try {
      const res = await fetch(`${process.env.API_ENDPOINT}/apps`, {
        headers: {
          "Content-Type": "application/json",
        },
        method: "GET",
        credentials: "include",
      });
      if (!res.ok) {
        if (res.status === 401) {
          Utilities.logoutUser();
          return null;
        }
        throw Error(`Failed to fetch apps with status ${res.status}`);
      }
      const { apps } = await res.json();

      // adding labels to apps list so <Select> can render them
      const appsWithLabels = await apps?.map((app) => ({
        ...app,
        value: app.name,
        label: app.name,
      }));
      if (isEmpty(selectedApp)) {
        setAppsList(appsWithLabels);
        setSelectedApp(appsWithLabels[0]);
      } else {
        const updateSelectedApp = appsWithLabels.find((app) => {
          return app.id === selectedApp?.id;
        });
        setAppsList(appsWithLabels);
        setSelectedApp(updateSelectedApp);
        getInitialOwnerRepo(updateSelectedApp);
        setGitopsEnabled(updateSelectedApp?.downstream.gitops?.enabled);
        setGitopsConnected(updateSelectedApp?.downstream.gitops?.isConnected);
      }
    } catch (err) {
      throw Error(err);
    }
  };

  const getGitops = async () => {
    await fetchGitops();
    if (freshGitops?.enabled) {
      getInitialOwnerRepo(selectedApp);
      const foundSelectedService = find(
        SERVICES,
        (service) => service.value === freshGitops.provider
      );
      setSelectedService(
        foundSelectedService ? foundSelectedService : selectedService
      );
      setHostname(freshGitops.hostname || "");
      setHttpPort(freshGitops.httpPort || "");
      setSshPort(freshGitops.sshPort || "");
      setGitops(freshGitops);
    } else {
      setGitops(freshGitops);
    }
  };

  useEffect(() => {
    getAppsList();
  }, []);

  useEffect(() => {
    // will run whenever selectedApp changes
    getGitops();
  }, [selectedApp]);

  const handleServiceChange = (service) => {
    setSelectedService(service);
  };

  const getGitOpsInput = (uri, tempBranch, tempPath, format, action) => {
    let gitOpsInput = new Object();
    gitOpsInput.provider = provider;
    gitOpsInput.uri = uri;
    gitOpsInput.branch = tempBranch || "";
    gitOpsInput.path = tempPath;
    gitOpsInput.format = format;
    gitOpsInput.action = action;

    if (requiresHostname(provider)) {
      gitOpsInput.hostname = hostname;
    }

    const isBitbucketServer = provider === "bitbucket_server";
    if (isBitbucketServer) {
      gitOpsInput.httpPort = httpPort;
      gitOpsInput.sshPort = sshPort;
    }

    return gitOpsInput;
  };

  const providerChanged = () => {
    return selectedService?.value !== gitops?.provider;
  };

  const resetGitOps = async () => {
    const res = await fetch(`${process.env.API_ENDPOINT}/gitops/reset`, {
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
      method: "POST",
    });
    if (!res.ok) {
      if (res.status === 401) {
        Utilities.logoutUser();
        return;
      }
      throw new Error(`Unexpected status code: ${res.status}`);
    }
  };

  const createGitOpsRepo = async (gitOpsInput) => {
    const res = await fetch(`${process.env.API_ENDPOINT}/gitops/create`, {
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
      body: JSON.stringify({
        gitOpsInput: gitOpsInput,
      }),
      method: "POST",
    });
    if (!res.ok) {
      if (res.status === 401) {
        Utilities.logoutUser();
        return;
      }
      throw new Error(`Unexpected status code: ${res.status}`);
    }
  };

  const updateAppGitOps = async (appId, clusterId, gitOpsInput) => {
    const res = await fetch(
      `${process.env.API_ENDPOINT}/gitops/app/${appId}/cluster/${clusterId}/update`,
      {
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          gitOpsInput: gitOpsInput,
        }),
        method: "PUT",
      }
    );
    if (!res.ok) {
      if (res.status === 401) {
        Utilities.logoutUser();
        return;
      }
      throw new Error(`Unexpected status code: ${res.status}`);
    }
  };

  const finishSetup = async (repoDetails = {}) => {
    setFinishingSetup(true);
    setErrorTitle("");
    setErrorMsg("");
    setDisplayErrorModal(false);

    const {
      ownerRepo = "",
      tempBranch = branch,
      tempPath = path,
      action = "commit",
      format = "single",
    } = repoDetails;

    const newHttpPort = httpPort || BITBUCKET_SERVER_DEFAULT_HTTP_PORT;
    const newSshPort = sshPort || BITBUCKET_SERVER_DEFAULT_SSH_PORT;

    const repoUri = getGitOpsUri(provider, ownerRepo, hostname, httpPort);
    const gitOpsInput = getGitOpsInput(
      repoUri,
      tempBranch,
      tempPath,
      format,
      action,
      newHttpPort,
      newSshPort
    );
    try {
      if (gitops?.enabled && providerChanged()) {
        await resetGitOps();
      }
      await createGitOpsRepo(gitOpsInput);

      const currentApp = find(appsList, {
        id: selectedApp.id,
      });

      const downstream = currentApp?.downstream;
      const clusterId = downstream?.cluster?.id;

      await updateAppGitOps(currentApp.id, clusterId, gitOpsInput);
      await getAppsList();

      return true;
    } catch (err) {
      console.log(err);
      setErrorTitle("Failed to finish gitops setup");
      setErrorMsg(
        err ? err.message : "Something went wrong, please try again."
      );
      setDisplayErrorModal(true);

      return false;
    } finally {
      setFinishingSetup(false);
    }
  };

  const validStep = (stepToValidate) => {
    setProviderError(null);
    if (stepToValidate === "provider") {
      if (requiresHostname(provider) && !hostname.length) {
        setProviderError({ field: "hostname" });
        return false;
      }
    }
    return true;
  };

  const stepFrom = (from, to) => {
    if (validStep(from)) {
      setStep(to);
    }
  };

  const toggleErrorModal = () => {
    setDisplayErrorModal(!displayErrorModal);
  };

  const handleAppChange = (app) => {
    const currentApp = find(appsList, { id: app.id });
    getInitialOwnerRepo(currentApp);
    setSelectedApp(app);
  };

  const isSingleApp = appsList?.length === 1;

  return (
    <GitOpsContext.Provider
      value={{
        step,
        setStep,
        hostname,
        setHostname,
        httpPort,
        setHttpPort,
        sshPort,
        setSshPort,
        services,
        setServices,
        selectedService,
        setSelectedService,
        providerError,
        setProviderError,
        finishingSetup,
        setFinishingSetup,
        appsList,
        setAppsList,
        gitops,
        setGitops,
        errorMsg,
        setErrorMsg,
        errorTitle,
        setErrorTitle,
        displayErrorModal,
        setDisplayErrorModal,
        toggleErrorModal,
        selectedApp,
        setSelectedApp,
        owner,
        setOwner,
        repo,
        setRepo,
        branch,
        setBranch,
        path,
        setPath,
        gitopsConnected,
        setGitopsConnected,
        gitopsEnabled,
        setGitopsEnabled,
        handleServiceChange,
        finishSetup,
        handleAppChange,
        getAppsList,
        getGitops,
        isSingleApp,
        provider,
        stepFrom,
      }}
    >
      {children}
    </GitOpsContext.Provider>
  );
};

const GitOpsConsumer = GitOpsContext.Consumer;
export function withGitOpsConsumer(Component) {
  return function ConsumerWrapper(props) {
    return (
      <GitOpsConsumer>
        {/*  returning the component that was passed in , access the possible props */}
        {(value) => <Component {...props} context={value} />}
      </GitOpsConsumer>
    );
  };
}

export { GitOpsProvider, GitOpsConsumer, GitOpsContext };
