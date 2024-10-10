import { useContext, useCallback, useEffect, useState } from "react";
import { Utilities } from "../../../utilities/utilities";
import DisableModal from "./modals/DisableModal";
import { GitOpsContext, withGitOpsConsumer } from "../context";
import AppSelector from "./AppSelector";
import { getLabel, addLabelsToApps } from "../utils";
import GitOpsProviderSelector from "./GitOpsProviderSelector";
import GitOpsRepoDetails from "./GitOpsRepoDetails";
import { updateAppsList } from "../utils";
import { KotsPageTitle } from "@components/Head";

const SetupProvider = ({ appName }) => {
  const {
    handleAppChange,
    selectedApp,
    step,
    appsList,
    getAppsList,
    isSingleApp,
    provider,
    gitopsEnabled,
    gitopsConnected,
  } = useContext(GitOpsContext);
  const [app, setApp] = useState({});

  // something funky is happening here, if we use appsList from context,
  // the app selector hover state gets messed up
  // this is just a temp fix
  const apps = appsList?.map((app) => ({
    ...app,
    value: app.name,
    label: app.name,
  }));

  useEffect(() => {
    if (appsList.length > 0) {
      setApp(
        apps.find((app) => {
          return app.id === selectedApp?.id;
        })
      );
    }
  }, [selectedApp, appsList]);

  const [showDisableGitopsModalPrompt, setShowDisableGitopsModalPrompt] =
    useState(false);
  const [disablingGitOps, setDisablingGitOps] = useState(false);

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
            "Content-Type": "application/json",
          },
          credentials: "include",
          method: "POST",
        }
      );
      if (!res.ok && res.status === 401) {
        Utilities.logoutUser();
        return;
      }
      if (res.ok && res.status === 204) {
        await getAppsList();

        setShowDisableGitopsModalPrompt(false);
      }
    } catch (err) {
      console.log(err);
    } finally {
      setDisablingGitOps(false);
    }
  };

  return (
    <div
      key={`${step}-active`}
      className="GitOpsDeploy--step card-bg u-textAlign--left"
    >
      <KotsPageTitle pageName="GitOps Configuration" />
      <p className="step-title card-title">GitOps Configuration</p>
      <p className="step-sub">
        Connect a git version control system so all application updates are
        committed to a git repository.
        <br /> When GitOps is enabled, you cannot deploy updates directly from
        the Admin Console.
      </p>
      <div className="flex-column u-textAlign--left card-item u-padding--15">
        <div className="flex alignItems--center u-marginBottom--30">
          {isSingleApp && app ? (
            <div className="u-marginRight--5">{getLabel(app)}</div>
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
                disabled={disablingGitOps}
                onClick={promptToDisableGitOps}
                className="link u-fontWeight--normal"
              >
                {disablingGitOps
                  ? "Disabling GitOps"
                  : "Disable GitOps for this app"}
              </a>
            )}
          </div>
        </div>
        <GitOpsProviderSelector />
        <GitOpsRepoDetails />
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

export default withGitOpsConsumer(SetupProvider);
