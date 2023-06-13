import React, { useEffect, useReducer } from "react";
import { Link, useParams, useNavigate } from "react-router-dom";
import { KotsPageTitle } from "@components/Head";
import AceEditor, { Marker } from "react-ace";
import "brace/mode/text";
import "brace/mode/yaml";
import "brace/theme/chrome";

import Loader from "../shared/Loader";

import "../../scss/components/redactors/EditRedactor.scss";
import Icon from "../Icon";
import { useSelectedApp } from "@features/App";

type State = {
  activeMarkers?: Marker[];
  createConfirm: boolean;
  createErrMsg: string;
  creatingRedactor: boolean;
  editConfirm: boolean;
  editingErrMsg: string;
  editingRedactor: boolean;
  isLoadingRedactor: boolean;
  redactorEnabled: boolean;
  redactorErrMsg: string;
  redactorName: string;
  redactorYaml: string;
};

// TODO: upgrade AceEditor so we can use the type definitions
let aceEditor: AceEditor | null;

const EditRedactor = () => {
  const [state, setState] = useReducer(
    (currentState: State, newState: Partial<State>) => ({
      ...currentState,
      ...newState,
    }),
    {
      createConfirm: false,
      createErrMsg: "",
      creatingRedactor: false,
      editConfirm: false,
      editingErrMsg: "",
      editingRedactor: false,
      isLoadingRedactor: false,
      redactorEnabled: false,
      redactorErrMsg: "",
      redactorName: "",
      redactorYaml: "",
    }
  );

  const navigate = useNavigate();
  const params = useParams();
  const slug = useSelectedApp()?.slug || "";

  const getRedactor = (redactorSlug: string) => {
    setState({
      isLoadingRedactor: true,
      redactorErrMsg: "",
    });

    fetch(`${process.env.API_ENDPOINT}/redact/spec/${redactorSlug}`, {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
    })
      .then((res) => res.json())
      .then((result) => {
        if (result.success) {
          setState({
            redactorYaml: result.redactor,
            redactorName: result.redactorMetadata.name,
            redactorEnabled: result.redactorMetadata.enabled,
            isLoadingRedactor: false,
            redactorErrMsg: "",
          });
        } else {
          setState({
            isLoadingRedactor: false,
            redactorErrMsg: result.error,
          });
        }
      })
      .catch((err) => {
        setState({
          isLoadingRedactor: false,
          redactorErrMsg: err,
        });
      });
  };

  const editRedactor = (
    redactorSlug: string,
    enabled: boolean,
    yaml: string
  ) => {
    setState({ editingRedactor: true, editingErrMsg: "" });

    const payload = {
      enabled: enabled,
      redactor: yaml,
    };

    fetch(`${process.env.API_ENDPOINT}/redact/spec/${redactorSlug}`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
      body: JSON.stringify(payload),
    })
      .then(async (res) => {
        const editResponse = await res.json();
        if (!res.ok) {
          setState({
            editingRedactor: false,
            editingErrMsg: editResponse.error,
          });
          return;
        }

        if (editResponse.success) {
          setState({
            redactorYaml: editResponse.redactor,
            redactorName: editResponse.redactorMetadata.name,
            redactorEnabled: editResponse.redactorMetadata.enabled,
            editingRedactor: false,
            editConfirm: true,
            createErrMsg: "",
          });
          setTimeout(() => {
            setState({ editConfirm: false });
          }, 3000);
          navigate(`/app/${redactorSlug}/troubleshoot/redactors`, {
            replace: true,
          });
        } else {
          setState({
            editingRedactor: false,
            editingErrMsg: editResponse.error,
          });
        }
      })
      .catch((err) => {
        setState({
          editingRedactor: false,
          editingErrMsg: err.message
            ? err.message
            : "Something went wrong, please try again.",
        });
      });
  };

  const getEmptyNameLine = (redactorYaml: string) => {
    const splittedYaml = redactorYaml.split("\n");
    let metadataFound = false;
    let namePosition;
    for (let i = 0; i < splittedYaml.length; ++i) {
      if (splittedYaml[i] === "metadata:") {
        metadataFound = true;
      }
      if (metadataFound && splittedYaml[i].includes("name:")) {
        namePosition = i + 1;
        break;
      }
    }
    return namePosition;
  };

  const createRedactor = (
    enabled: boolean,
    newRedactor: boolean,
    yaml: string
  ) => {
    setState({ creatingRedactor: true, createErrMsg: "" });

    const payload = {
      enabled: enabled,
      new: newRedactor,
      redactor: yaml,
    };

    fetch(`${process.env.API_ENDPOINT}/redact/spec/new`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
      body: JSON.stringify(payload),
    })
      .then(async (res) => {
        const createResponse = await res.json();
        if (!res.ok) {
          setState({
            creatingRedactor: false,
            createErrMsg: createResponse.error,
          });
          // TODO: fix after upgradig AceEditor
          // eslint-disable-next-line
          // @ts-ignore
          const editor = aceEditor.editor;

          editor.scrollToLine(getEmptyNameLine(state.redactorYaml), true, true);
          editor.gotoLine(getEmptyNameLine(state.redactorYaml), 1, true);
        }

        if (createResponse.success) {
          setState({
            redactorYaml: createResponse.redactor,
            redactorName: createResponse.redactorMetadata.name,
            redactorEnabled: createResponse.redactorMetadata.enabled,
            creatingRedactor: false,
            createConfirm: true,
            createErrMsg: "",
          });
          setTimeout(() => {
            setState({ createConfirm: false });
          }, 3000);
          navigate(`/app/${slug}/troubleshoot/redactors`, { replace: true });
        } else {
          setState({
            creatingRedactor: false,
            createErrMsg: createResponse.error,
          });
        }
      })
      .catch((err) => {
        setState({
          creatingRedactor: false,
          createErrMsg: err.message
            ? err.message
            : "Something went wrong, please try again.",
        });
      });
  };

  const handleEnableRedactor = () => {
    setState({
      redactorEnabled: !state.redactorEnabled,
    });
  };

  useEffect(() => {
    if (params.redactorSlug) {
      getRedactor(params.redactorSlug);
    } else {
      const defaultYaml = `kind: Redactor
apiVersion: troubleshoot.sh/v1beta2
metadata:
  name:
spec:
  redactors:
  - name: myredactor
    fileSelector:
      files:
      - "abc"
    removals:
      values:
      - "removethis"`;
      setState({
        redactorEnabled: true,
        redactorYaml: defaultYaml,
        redactorName: "New redactor",
      });
    }
  }, []);

  const onYamlChange = (value: string) => {
    setState({ redactorYaml: value });
  };

  const onSaveRedactor = () => {
    if (params.redactorSlug) {
      editRedactor(
        params.redactorSlug,
        state.redactorEnabled,
        state.redactorYaml
      );
    } else {
      createRedactor(state.redactorEnabled, true, state.redactorYaml);
    }
  };

  const {
    isLoadingRedactor,
    createConfirm,
    editConfirm,
    creatingRedactor,
    editingRedactor,
    createErrMsg,
    editingErrMsg,
  } = state;

  if (isLoadingRedactor) {
    return (
      <div className="flex-column flex1 alignItems--center justifyContent--center">
        <Loader size="60" />
      </div>
    );
  }

  return (
    <div className="container flex-column flex1 u-overflow--auto u-paddingTop--30 u-paddingBottom--20 justifyContent--center alignItems--center">
      <KotsPageTitle pageName="New Redactor" showAppSlug />
      <div className="Redactors--wrapper flex1 flex-column u-width--full">
        {(createErrMsg || editingErrMsg) && (
          <p className="ErrorToast flex justifyContent--center alignItems--center">
            {createErrMsg ? createErrMsg : editingErrMsg}
          </p>
        )}
        <div className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy u-marginBottom--20">
          <Link
            to={`/app/${slug}/troubleshoot/redactors`}
            className="link u-marginRight--5"
          >
            Redactors
          </Link>{" "}
          &gt; <span className="u-marginLeft--5">{state.redactorName}</span>
        </div>
        <div className="flex flex-auto alignItems--flexStart justifyContent--spaceBetween">
          <div className="flex flex1 alignItems--center">
            <p className="u-fontWeight--bold u-textColor--primary u-fontSize--jumbo u-lineHeight--normal u-marginRight--10">
              {state.redactorName}
            </p>
          </div>
          <div className="flex justifyContent--flexEnd">
            <div className="toggle flex flex1">
              <div className="flex flex1">
                <div
                  className={`Checkbox--switch ${
                    state.redactorEnabled ? "is-checked" : "is-notChecked"
                  }`}
                >
                  <input
                    type="checkbox"
                    className="Checkbox-toggle"
                    name="isRedactorEnabled"
                    checked={state.redactorEnabled}
                    onChange={() => {
                      handleEnableRedactor();
                    }}
                  />
                </div>
              </div>
              <div className="flex flex1 u-marginLeft--5">
                <p className="u-fontWeight--medium u-textColor--secondary u-fontSize--large alignSelf--center">
                  {state.redactorEnabled ? "Enabled" : "Disabled"}
                </p>
              </div>
            </div>
          </div>
        </div>
        <p className="u-fontSize--normal u-textColor--bodyCopy u-fontWeight--medium u-lineHeight--normal u-marginTop--10">
          For more information about creating redactors,
          <a
            href="https://troubleshoot.sh/reference/redactors/overview/"
            target="_blank"
            rel="noopener noreferrer"
            className="link"
          >
            {" "}
            check out our docs
          </a>
          .
        </p>
        <div className="flex1 u-marginTop--30 u-border--gray">
          <AceEditor
            ref={(el) => (aceEditor = el)}
            mode="yaml"
            theme="chrome"
            className="flex1 flex"
            value={state.redactorYaml}
            height="100%"
            width="100%"
            markers={state.activeMarkers}
            editorProps={{
              $blockScrolling: Infinity,
              // @ts-ignore
              useSoftTabs: true,
              tabSize: 2,
            }}
            onChange={(value) => onYamlChange(value)}
            setOptions={{
              scrollPastEnd: false,
              showGutter: true,
            }}
          />
        </div>
        <div className="flex u-marginTop--20 justifyContent--spaceBetween">
          <div className="flex">
            <Link
              to={`/app/${slug}/troubleshoot/redactors`}
              className="btn secondary"
            >
              {" "}
              Cancel{" "}
            </Link>
          </div>
          <div className="flex alignItems--center">
            {createConfirm ||
              (editConfirm && (
                <div className="u-marginRight--10 flex alignItems--center">
                  <Icon
                    icon="check-circle-filled"
                    size={16}
                    className="success-color"
                  />
                  <span className="u-marginLeft--5 u-fontSize--small u-fontWeight--medium u-textColor--success">
                    {createConfirm ? "Redactor created" : "Redactor updated"}
                  </span>
                </div>
              ))}
            <button
              type="button"
              className="btn primary blue"
              onClick={onSaveRedactor}
              disabled={creatingRedactor || editingRedactor}
            >
              {creatingRedactor || editingRedactor ? "Saving" : "Save redactor"}{" "}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default EditRedactor;
