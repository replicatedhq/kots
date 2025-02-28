import Modal from "react-modal";
import MonacoEditor from "@monaco-editor/react";

import Loader from "../shared/Loader";

export default function ViewSnapshotLogsModal(props) {
  const {
    displayShowSnapshotLogsModal,
    toggleViewLogsModal,
    logs,
    snapshotDetails,
    loadingSnapshotLogs,
    snapshotLogsErr,
    snapshotLogsErrMsg,
  } = props;

  return (
    <Modal
      isOpen={displayShowSnapshotLogsModal}
      onRequestClose={toggleViewLogsModal}
      shouldReturnFocusAfterClose={false}
      contentLabel="Snapshot logs"
      ariaHideApp={false}
      className="Modal FullSize"
    >
      <div className="Modal-body flex1 flex-column" style={{ height: "97%" }} data-testid="snapshot-logs-modal">
        <p className="u-fontSize--larger u-fontWeight--bold u-textColor--primary u-marginBottom--5">
          {snapshotDetails?.name} logs
        </p>
        <div className="flex1 flex-column u-position--relative u-marginTop--10">
          {loadingSnapshotLogs ? (
            <div className="flex-column flex1 alignItems--center justifyContent--center">
              <Loader size="60" />
            </div>
          ) : snapshotLogsErr ? (
            <div className="flex1 flex-column justifyContent--center alignItems--center">
              <span className="icon redWarningIcon" />
              <p className="u-textColor--error u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-marginTop--10">
                {snapshotLogsErrMsg}
              </p>
            </div>
          ) : (
            <MonacoEditor
              value={logs}
              height="100%"
              width="100%"
              language="bash"
              options={{
                readOnly: true,
                contextmenu: false,
                minimap: {
                  enabled: false,
                },
                scrollBeyondLastLine: false,
              }}
            />
          )}
        </div>

        <div className="u-marginTop--10 flex">
          <button
            onClick={() => toggleViewLogsModal()}
            className="btn primary blue"
          >
            Ok, got it!
          </button>
        </div>
      </div>
    </Modal>
  );
}
