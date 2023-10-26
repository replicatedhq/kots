import { useState, useContext } from "react";
import { GitOpsContext, withGitOpsConsumer } from "../context";
import { Flex, Paragraph } from "../../../styles/common";
import Loader from "../../../components/shared/Loader";
import { usePrevious } from "../../../hooks/usePrevious";

const GitopsRepoDetails = () => {
  const {
    selectedService,
    owner,
    setOwner,
    repo,
    setRepo,
    branch,
    setBranch,
    path,
    setPath,
    finishingSetup,
    setFinishingSetup,
    finishSetup,
    gitopsConnected,
    gitopsEnabled,
    providerError,
    setProviderError,
    stepFrom,
  } = useContext(GitOpsContext);
  const [action, setAction] = useState("commit");
  const [format, setFormat] = useState("single");
  const previousOwner = usePrevious(owner);
  const previousRepo = usePrevious(repo);
  const previousBranch = usePrevious(branch);
  const previousPath = usePrevious(path);
  const provider = selectedService?.value;
  const isBitbucketServer = provider === "bitbucket_server";

  const isValid = () => {
    if (provider !== "other" && !owner.length) {
      setProviderError({ field: "owner" });
      return false;
    }
    return true;
  };

  const onFinishSetup = async () => {
    if (!isValid() || !finishSetup) {
      return;
    }

    setFinishingSetup(true);
    const ownerRepo = owner + "/" + repo;

    const repoDetails = {
      ownerRepo: ownerRepo,
      branch: branch,
      path: path,
      action: action,
      format: format,
    };

    const success = await finishSetup(repoDetails);

    if (success) {
      setFinishingSetup(false);
      stepFrom("provider", "action");
    } else {
      setFinishingSetup(false);
    }
  };

  const allowUpdate = () => {
    if (provider === "other") {
      return true;
    }
    if (!gitopsEnabled || !gitopsConnected) {
      return true;
    } else if (
      owner !== previousOwner ||
      repo !== previousRepo ||
      branch !== previousBranch ||
      path !== previousPath
    ) {
      return true;
    }

    return false;
  };

  return (
    <>
      <Flex key={`action-active`} width="100%" direction="column">
        <Flex flex="1" mt="30" mb="20" width="100%">
          <div className="flex flex1 flex-column u-marginRight--20">
            <p className="card-item-title">
              {isBitbucketServer ? "Project" : "Owner"}
              <span className="card-item-title"> (Required)</span>
            </p>
            <input
              type="text"
              className={`Input ${
                providerError?.field === "owner" && "has-error"
              }`}
              placeholder={isBitbucketServer ? "project" : "owner"}
              value={owner}
              onChange={(e) => setOwner(e.target.value)}
            />
            {providerError?.field === "owner" && (
              <p className="u-fontSize--small u-marginTop--5 u-color--chestnut u-fontWeight--medium u-lineHeight--normal">
                {isBitbucketServer
                  ? "A project must be provided"
                  : "An owner must be provided"}
              </p>
            )}
          </div>
          <Flex flex="1" direction="column">
            <p className="card-item-title">
              Repository <span className="card-item-title">(Required)</span>
            </p>
            <input
              type="text"
              className={`Input ${
                providerError?.field === "repo" && "has-error"
              }`}
              placeholder={"Repository"}
              value={repo}
              onChange={(e) => setRepo(e.target.value)}
            />
            {providerError?.field === "owner" && (
              <p className="u-fontSize--small u-marginTop--5 u-color--chestnut u-fontWeight--medium u-lineHeight--normal">
                A repository must be provided
              </p>
            )}
          </Flex>
        </Flex>

        <Flex width="100%">
          <div className="flex flex1 flex-column u-marginRight--20">
            <p className="card-item-title">Branch</p>
            <p className="u-fontSize--normal help-text-color u-fontWeight--medium u-lineHeight--normal u-marginBottom--10">
              Leave blank to use the default branch.
            </p>
            <input
              type="text"
              className={`Input`}
              placeholder="main"
              value={branch}
              onChange={(e) => setBranch(e.target.value)}
            />
          </div>
          <div className="flex flex1 flex-column">
            <p className="card-item-title">Path</p>
            <p className="u-fontSize--normal help-text-color u-fontWeight--medium u-lineHeight--normal u-marginBottom--10">
              Path in repository to commit deployment file
            </p>
            <input
              type="text"
              className={"Input"}
              placeholder="/path/to-deployment"
              value={path}
              onChange={(e) => setPath(e.target.value)}
            />
          </div>
        </Flex>
      </Flex>
      <div
        className="flex justifyContent--flexEnd u-marginTop--30"
        style={{ width: "100%" }}
      >
        {finishingSetup ? (
          <Loader className="u-marginLeft--5" size="30" />
        ) : gitopsConnected && gitopsEnabled ? (
          <button
            className="btn primary blue"
            type="button"
            disabled={finishingSetup || !allowUpdate()}
            onClick={onFinishSetup}
          >
            Save Configuration
          </button>
        ) : (
          <button
            className="btn primary blue"
            type="button"
            disabled={finishingSetup || !allowUpdate()}
            onClick={onFinishSetup}
          >
            Generate SSH key
          </button>
        )}
      </div>
    </>
  );
};

export default withGitOpsConsumer(GitopsRepoDetails);
