import React from "react";
import Modal from "react-modal";
import AceEditor from "react-ace";

import Loader from "../shared/Loader";

export default function ViewSnapshotLogsModal(props) {
  const { displayShowSnapshotLogsModal, toggleViewLogsModal, logs, snapshotDetails, loadingSnapshotLogs } = props;

  return (
    <Modal
      isOpen={displayShowSnapshotLogsModal}
      onRequestClose={toggleViewLogsModal}
      shouldReturnFocusAfterClose={false}
      contentLabel="Snapshot logs"
      ariaHideApp={false}
      className="Modal FullSize"
    >
      <div className="Modal-body flex1 flex-column" style={{ height: "97%" }}>
        <p className="u-fontSize--larger u-fontWeight--bold u-color--tuna u-marginBottom--5">{snapshotDetails?.name} logs</p>
        <div className="flex1 flex-column u-position--relative u-marginTop--10">
          {loadingSnapshotLogs ?
            <div className="flex-column flex1 alignItems--center justifyContent--center">
              <Loader size="60" />
            </div>
            :
            <AceEditor
              value={logs}
              theme="chrome"
              className="flex1 flex"
              height="100%"
              width="100%"
              readOnly={true}
              editorProps={{
                $blockScrolling: true,
                useSoftTabs: true,
                tabSize: 2,
              }}
            />}
        </div>

        <div className="u-marginTop--10 flex">
          <button onClick={() => toggleViewLogsModal()} className="btn primary blue">Ok, got it!</button>
        </div>
      </div>
    </Modal>
  );
}