import { useEffect, useReducer, useState } from "react";
import { useParams } from "react-router-dom";
import MonacoEditor from "@monaco-editor/react";
import CodeSnippet from "./shared/CodeSnippet";
import ErrorModal from "./modals/ErrorModal";
import { useSelectedApp } from "@features/App";
import "../scss/components/PreflightCheckPage.scss";

import { KotsParams } from "@types";
import { PreflightResult } from "@src/features/PreflightChecks/types";

type Props = {
  errors?: string[];
  ignorePermissionErrors: () => void;
  logo: string;
  preflightResultData?: PreflightResult[];
  showRbacError: boolean;
};

type State = {
  command: string | null;
  displayErrorModal: boolean;
  errorTitle: string;
  errorMsg: string;
  showErrorDetails: boolean;
};

const fetchPreflightCommand = async (slug: string, sequence: string) => {
  const res = await fetch(
    `${process.env.API_ENDPOINT}/app/${slug}/sequence/${sequence}/preflightcommand`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        origin: window.location.origin,
      }),
      credentials: "include",
    }
  );
  if (!res.ok) {
    throw new Error(`Unexpected status code: ${res.status}`);
  }
  const response = await res.json();
  return response.command;
};

const PreflightResultErrors = (props: Props) => {
  const [state, setState] = useReducer(
    (currentState: State, newState: Partial<State>) => ({
      ...currentState,
      ...newState,
    }),
    {
      command: null,
      showErrorDetails: false,
      errorTitle: "",
      errorMsg: "",
      displayErrorModal: false,
    }
  );
  const { slug, sequence = "0" } = useParams<keyof KotsParams>() as KotsParams;
  const selectedApp = useSelectedApp();

  const [previousAppSlug, setPreviousAppSlug] = useState(slug);
  const [previousSequence, setPreviousSequence] = useState<string | undefined>(
    sequence
  );

  const getPreflightCommand = async () => {
    try {
      const command = await fetchPreflightCommand(slug, sequence);
      setState({
        command,
      });
    } catch (err) {
      if (err instanceof Error) {
        setState({
          errorTitle: `Failed to get preflight command`,
          errorMsg: err
            ? err.message
            : "Something went wrong, please try again.",
          displayErrorModal: true,
        });
        return;
      }
      setState({
        errorTitle: `Failed to get preflight command`,
        errorMsg: "Something went wrong, please try again.",
        displayErrorModal: true,
      });
    }
  };

  useEffect(() => {
    if (!props.preflightResultData) {
      return;
    }
    getPreflightCommand();
  }, []);

  useEffect(() => {
    if (!props.preflightResultData) {
      return;
    }

    // TODO: determine if it's actually necessary to track the previous props
    if (previousAppSlug !== slug || previousSequence !== sequence) {
      getPreflightCommand();
    }
    setPreviousAppSlug(slug);
    setPreviousSequence(sequence);
  }, [props.preflightResultData]);

  const toggleShowErrorDetails = () => {
    setState({
      showErrorDetails: !state.showErrorDetails,
    });
  };

  const toggleErrorModal = () => {
    setState({ displayErrorModal: !state.displayErrorModal });
  };

  const { errors, logo } = props;
  const { errorTitle, errorMsg, displayErrorModal, command } = state;

  const displayErrorString = errors !== undefined ? errors.join("\n") : "";

  return (
    <div className="flex flex1 flex-column">
      <div className="flex flex1 u-height--full u-width--full u-marginTop--5 tw-mb-6">
        <div className="flex-column u-width--full u-overflow--hidden u-paddingTop--30 u-paddingBottom--5 alignItems--center justifyContent--center">
          <div className="PreChecksBox-wrapper flex-column u-padding--20">
            <div className="flex">
              {logo && (
                <div className="flex-auto u-marginRight--10">
                  <div
                    className="watch-icon"
                    style={{
                      backgroundImage: `url(${logo})`,
                      width: "36px",
                      height: "36px",
                    }}
                  ></div>
                </div>
              )}
              <h2 className="u-fontSize--largest u-textColor--primary u-fontWeight--bold u-lineHeight--normal">
                Unable to automatically run preflight checks
              </h2>
            </div>
            {props.showRbacError && (
              <p className="u-marginTop--10 u-marginBottom--10 u-fontSize--normal u-lineHeight--normal u-textColor--bodyCopy u-fontWeight--normal">
                The Kubernetes RBAC policy that the Admin Console is running
                with does not have access to complete the Preflight Checks. It’s
                recommended that you run these manually before proceeding.
              </p>
            )}
            {props.showRbacError && (
              <p className="u-marginTop--10 u-marginBottom--10 u-fontSize--normal u-lineHeight--normal u-textColor--bodyCopy u-fontWeight--normal">
                There were errors running preflight checks in Admin Console.
                Preflight checks can be ran manually as an alternative. It’s
                recommended that you run these before proceeding.
              </p>
            )}
            <p
              className="link u-fontSize--normal u-marginBottom--10"
              onClick={toggleShowErrorDetails}
            >
              {state.showErrorDetails ? "Hide details" : "Show details"}
            </p>
            {state.showErrorDetails && (
              <div className="flex-column flex flex1 monaco-editor-wrapper u-border--gray">
                <MonacoEditor
                  language="bash"
                  value={displayErrorString}
                  height="300px"
                  options={{
                    readOnly: true,
                    contextmenu: false,
                    minimap: {
                      enabled: false,
                    },
                    scrollBeyondLastLine: false,
                  }}
                />
              </div>
            )}
            <div className="u-marginTop--20">
              <h2 className="u-fontSize--largest u-textColor--primary u-fontWeight--bold u-lineHeight--normal">
                Run Preflight Checks Manually
              </h2>
              <p className="u-fontSize--normal u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--20">
                Run the commands below from your workstation to complete the
                Preflight Checks.
              </p>
              {command ? (
                <CodeSnippet
                  language="bash"
                  canCopy={true}
                  onCopyText={
                    <span className="u-textColor--success">
                      Command has been copied to your clipboard
                    </span>
                  }
                >
                  {command}
                </CodeSnippet>
              ) : null}
            </div>
          </div>
        </div>
      </div>

      {errorMsg && (
        <ErrorModal
          errorModal={displayErrorModal}
          toggleErrorModal={toggleErrorModal}
          err={errorTitle}
          errMsg={errorMsg}
          appSlug={selectedApp?.slug || ""}
        />
      )}
    </div>
  );
};

export default PreflightResultErrors;
