import React from "react";
import Modal from "react-modal";
import MonacoEditor from "react-monaco-editor";
import Loader from "../shared/Loader";

export default function ShowLogsModal(props) {
  const { showLogsModal, hideLogsModal, viewLogsErrMsg, logs, selectedTab, logsLoading, renderLogsTabs } = props;

  return (
    <Modal
    isOpen={showLogsModal}
    onRequestClose={hideLogsModal}
    shouldReturnFocusAfterClose={false}
    contentLabel="View logs"
    ariaHideApp={false}
    className="Modal logs-modal"
  >
    <div className="Modal-body flex flex1">
      {viewLogsErrMsg ?
        <div class="flex1 flex-column justifyContent--center alignItems--center">
          <span className="icon redWarningIcon" />
          <p className="u-textColor--error u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-marginTop--10">{viewLogsErrMsg}</p>
        </div>
        :
        !logs || !selectedTab || logsLoading ? (
          <div className="flex-column flex1 alignItems--center justifyContent--center">
            <Loader size="60" />
          </div>
        ) : (
            <div className="flex-column flex1">
              <div className="flex-column flex1">
                {!logs.renderError && renderLogsTabs}
                <div className="flex-column flex1 u-border--gray monaco-editor-wrapper">
                  <MonacoEditor
                    language="json"
                    value={logs.renderError || logs[selectedTab]}
                    height="100%"
                    width="100%"
                    options={{
                      readOnly: true,
                      contextmenu: false,
                      minimap: {
                        enabled: false
                      },
                      scrollBeyondLastLine: false,
                    }}
                  />
                </div>
              </div>
              <div className="u-marginTop--20 flex">
                <button type="button" className="btn primary" onClick={hideLogsModal}>Ok, got it!</button>
              </div>
            </div>
          )}
    </div>
  </Modal>
  );
}