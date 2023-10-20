import { useContext } from "react";
import find from "lodash/find";
import Loader from "../../components/shared/Loader";
import ErrorModal from "../../components/modals/ErrorModal";
import { requiresHostname } from "../../utilities/utilities";
import { Flex, Paragraph } from "../../styles/common";
import SetupProvider from "./components/SetupProvider";
import AppGitops from "./components/AppGitops";
import { GitOpsContext, withGitOpsConsumer } from "./context";
import {
  SERVICES,
  BITBUCKET_SERVER_DEFAULT_SSH_PORT,
  BITBUCKET_SERVER_DEFAULT_HTTP_PORT,
} from "./constants";
import "../../scss/components/gitops/GitOpsDeploymentManager.scss";

const STEPS = [
  {
    step: "provider",
    title: "GitOps Configuration",
  },
  {
    step: "action",
    title: "GitOps Configuration ",
  },
];

const GitOpsDeploymentManager = (props) => {
  const {
    appsList,
    errorMsg,
    errorTitle,
    displayErrorModal,
    toggleErrorModal,
    step,
  } = useContext(GitOpsContext);

  const renderActiveStep = (step) => {
    switch (step.step) {
      case "provider":
        return <SetupProvider />;
      case "action":
        return <AppGitops />;
      default:
        return (
          <div key={`default-active`} className="GitOpsDeploy--step">
            default
          </div>
        );
    }
  };

  if (!appsList.length) {
    return (
      <div className="flex-column flex1 alignItems--center justifyContent--center">
        <Loader size="60" />
      </div>
    );
  }

  const activeStep = find(STEPS, { step });
  return (
    <div className="GitOpsDeploymentManager--wrapper flex-column flex1">
      {renderActiveStep(activeStep)}
      {errorMsg && (
        <ErrorModal
          errorModal={displayErrorModal}
          toggleErrorModal={toggleErrorModal}
          err={errorTitle}
          errMsg={errorMsg}
        />
      )}
    </div>
  );
};

export default withGitOpsConsumer(GitOpsDeploymentManager);
