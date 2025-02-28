import Modal from "react-modal";
import Loader from "../shared/Loader";
import { Link } from "react-router-dom";
import MonacoEditor, { loader } from "@monaco-editor/react";
import * as monaco from "monaco-editor/esm/vs/editor/editor.api.js";

// configures MonacoEditor to load files from node_modules rather than from CDN
loader.config({ monaco });

export default function ShowLogsModal(props) {
  const {
    showLogsModal,
    hideLogsModal,
    viewLogsErrMsg,
    logs,
    selectedTab,
    logsLoading,
    renderLogsTabs,
    versionFailing,
    troubleshootUrl,
  } = props;

  return (
    <Modal
      isOpen={showLogsModal}
      onRequestClose={hideLogsModal}
      shouldReturnFocusAfterClose={false}
      contentLabel="View logs"
      ariaHideApp={false}
      className="Modal logs-modal"
    >
      <div className="Modal-body flex flex1" data-testid="deploy-logs-modal">
        {viewLogsErrMsg ? (
          <div className="flex1 flex-column justifyContent--center alignItems--center">
            <span className="icon redWarningIcon" />
            <p className="u-textColor--error u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-marginTop--10">
              {viewLogsErrMsg}
            </p>
          </div>
        ) : !logs || !selectedTab || logsLoading ? (
          <div className="flex-column flex1 alignItems--center justifyContent--c enter">
            <Loader size="60" />
          </div>
        ) : (
          <div className="flex-column flex1">
            <div className="flex-column flex1">
              {!logs.renderError && renderLogsTabs}
              <div className="flex-column flex1 u-border--gray monaco-editor-wrapper" data-testid="deploy-logs-modal-editor">
                <MonacoEditor
                  language="json"
                  value={logs.renderError || logs[selectedTab]}
                  options={{
                    readOnly: true,
                    contextmenu: false,
                    minimap: {
                      enabled: false,
                    },
                    scrollBeyondLastLine: false,
                    wrappingStrategy: "advanced",
                    wordWrap: "on",
                  }}
                />
              </div>
            </div>
            <div className="u-marginTop--20 flex">
              <button
                type="button"
                className="btn primary"
                onClick={hideLogsModal}
              >
                Ok, got it!
              </button>
              {versionFailing && (
                <Link
                  to={troubleshootUrl}
                  className="btn secondary blue u-marginLeft--10"
                >
                  {" "}
                  Troubleshoot{" "}
                </Link>
              )}
            </div>
          </div>
        )}
      </div>
    </Modal>
  );
}
