import Modal from "react-modal";
import Select from "react-select";

import CodeSnippet from "../shared/CodeSnippet";

import { Utilities } from "../../utilities/utilities";
import Icon from "@components/Icon";

export default function BackupRestoreModal(props) {
  const {
    veleroNamespace,
    isMinimalRBACEnabled,
    restoreSnapshotModal,
    toggleRestoreModal,
    snapshotToRestore,
    includedApps,
    selectedRestore,
    onChangeRestoreOption,
    selectedRestoreApp,
    onChangeRestoreApp,
    handleApplicationSlugChange,
    appSlugToRestore,
    appSlugMismatch,
    handlePartialRestoreSnapshot,
    restoringSnapshot,
    getLabel,
  } = props;

  let restoreCommand = `kubectl kots restore --from-backup ${snapshotToRestore?.name}`;
  if (isMinimalRBACEnabled) {
    restoreCommand += ` --velero-namespace ${veleroNamespace}`;
  }
  if (selectedRestore === "kotsadm") {
    restoreCommand += " --exclude-apps";
  }

  return (
    <Modal
      isOpen={restoreSnapshotModal}
      shouldReturnFocusAfterClose={false}
      onRequestClose={() => {
        toggleRestoreModal({});
      }}
      ariaHideApp={false}
      contentLabel="Modal"
      className="Modal ConfigureSnapshots"
    >
      <div className="Modal-body">
        <div className="flex flex-column">
          <div
            className="flex"
            style={{ alignItems: "center", justifyContent: "space-between" }}
          >
            <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--more">
              {" "}
              Restore from backup{" "}
            </p>
            <button
              style={{ border: "none", background: "none", cursor: "pointer" }}
            >
              <Icon icon="close" onClick={toggleRestoreModal} size={15} />
            </button>
          </div>

          <p className="u-fontSize--normal u-fontWeight--normal u-textColor--bodyCopy u-lineHeight--normal">
            Select the type of backup you want to perform. A full restore of the
            Admin Console, your application and its metadata, application config
            and your database or a partial restore of your application and its
            metadata. All data not backed up will be lost and replaced with data
            in this backup.
          </p>
          <div className="flex flex1 justifyContent--spaceBetween u-marginTop--20 SnapshotRow--wrapper">
            <div className="flex flex-column">
              <p className="u-fontSize--normal u-fontWeight--bold u-textColor--primary u-lineHeight--normal">
                {snapshotToRestore?.name}
              </p>
              <p className="u-fontSize--normal u-textColor--bodyCopy u-fontWeight--medium u-lineHeight--normal u-marginRight--20">
                {Utilities.dateFormat(
                  snapshotToRestore?.startedAt,
                  "MMM D YYYY @ hh:mm a z"
                )}
              </p>
            </div>
            <div className="flex flex1 justifyContent--flexEnd">
              <p className="u-fontSize--normal u-textColor--bodyCopy u-fontWeight--bold u-lineHeight--normal u-marginRight--30 justifyContent--center flex alignItems--center">
                <span className="icon snapshot-volume-size-icon" />{" "}
                {snapshotToRestore?.volumeSizeHuman}{" "}
              </p>
              <p className="u-fontSize--normal u-textColor--bodyCopy u-fontWeight--bold u-lineHeight--normal justifyContent--center flex alignItems--center u-marginRight--30">
                <span className="icon snapshot-volume-icon" />{" "}
                {snapshotToRestore?.volumeSuccessCount}/
                {snapshotToRestore?.volumeCount}
              </p>
              <p className="u-fontSize--normal u-textColor--bodyCopy u-fontWeight--bold u-lineHeight--normal justifyContent--center flex alignItems--center">
                <span className="icon snapshot-app-icon" />{" "}
                {includedApps?.length} application
                {includedApps?.length > 1 && "s"}
              </p>
            </div>
          </div>
          <p className="u-fontSize--large u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--more u-marginTop--30">
            {" "}
            Will this be a full or partial restore?{" "}
          </p>
          <p className="u-fontSize--normal u-fontWeight--normal u-textColor--bodyCopy u-lineHeight--normal">
            You can do a full restore of the application, Admin Console, and
            databases or you can do a partial restore of just your application
            and its metadata.
          </p>
          <div className="flex-column flex1 u-marginTop--15">
            <div
              className={`SelectRestore--wrapper flex alignItems--center u-marginBottom--15 ${
                selectedRestore === "full" && "is-selected"
              }`}
              onClick={() => onChangeRestoreOption("full")}
            >
              <input
                className="u-marginRight--20"
                type="radio"
                checked={selectedRestore === "full"}
              />
              <span className="flex-auto icon snapshot-full-restore-icon" />
              <div className="flex flex-column u-marginLeft--10">
                <p className="u-fontSize--normal u-fontWeight--medium u-textColor--primary u-lineHeight--normal">
                  {" "}
                  Full restore{" "}
                </p>
                <p className="u-fontSize--small u-fontWeight--normal u-textColor--bodyCopy u-lineHeight--normal">
                  {" "}
                  Admin Console &amp; application{" "}
                </p>
              </div>
            </div>
            <div
              className={`SelectRestore--wrapper flex alignItems--center u-marginBottom--15 ${
                selectedRestore === "partial" && "is-selected"
              }`}
              onClick={() => onChangeRestoreOption("partial")}
            >
              <input
                className="u-marginRight--20"
                type="radio"
                checked={selectedRestore === "partial"}
              />
              <span className="flex-auto icon snapshot-partial-restore-icon u-marginRight--5" />
              <div className="flex flex-column u-marginLeft--10">
                <p className="u-fontSize--normal u-fontWeight--medium u-textColor--primary u-lineHeight--normal">
                  {" "}
                  Partial restore{" "}
                </p>
                <p className="u-fontSize--small u-fontWeight--normal u-textColor--bodyCopy u-lineHeight--normal">
                  {" "}
                  Application &amp; metadata only
                </p>
              </div>
            </div>
            <div
              className={`SelectRestore--wrapper flex alignItems--center ${
                selectedRestore === "kotsadm" && "is-selected"
              }`}
              onClick={() => onChangeRestoreOption("kotsadm")}
            >
              <input
                className="u-marginRight--20"
                type="radio"
                checked={selectedRestore === "kotsadm"}
              />
              <span className="flex-auto icon snapshot-kotsadm-restore-icon" />
              <div className="flex flex-column u-marginLeft--10">
                <p className="u-fontSize--normal u-fontWeight--medium u-textColor--primary u-lineHeight--normal">
                  {" "}
                  Restore Admin Console{" "}
                </p>
                <p className="u-fontSize--small u-fontWeight--normal u-textColor--bodyCopy u-lineHeight--normal">
                  {" "}
                  Only restores the Admin Console
                </p>
              </div>
            </div>
          </div>
          {selectedRestore === "full" || selectedRestore === "kotsadm" ? (
            <div className="flex flex-column u-marginTop--20">
              <p className="u-fontSize--large u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--more">
                {" "}
                To start the restore, run this command on your cluster.{" "}
              </p>
              <CodeSnippet
                key={restoreCommand}
                language="bash"
                canCopy={true}
                onCopyText={
                  <span className="u-textColor--success">
                    Command has been copied to your clipboard
                  </span>
                }
              >
                {restoreCommand}
              </CodeSnippet>
            </div>
          ) : includedApps?.length === 1 ? (
            <div className="flex flex-column u-marginTop--20">
              <div className="flex flex1 justifyContent--spaceBetween SnapshotRow--wrapper">
                <div className="flex flex1 alignItems--center">
                  <span
                    className="app-icon"
                    style={{
                      marginRight: "0.5em",
                      backgroundImage: `url(${selectedRestoreApp?.iconUri})`,
                    }}
                  ></span>
                  <p className="u-fontSize--normal u-fontWeight--bold u-textColor--primary u-lineHeight--normal">
                    {selectedRestoreApp?.name}
                  </p>
                </div>
                <div className="flex flex1 justifyContent--flexEnd">
                  <p className="u-fontSize--normal u-textColor--bodyCopy u-fontWeight--medium u-lineHeight--normal justifyContent--center">
                    {" "}
                    Sequence {selectedRestoreApp?.sequence}{" "}
                  </p>
                </div>
              </div>
              {appSlugMismatch ? (
                <p className="u-textColor--error u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginTop--10">
                  The app slug you entered does not match the current app slug
                </p>
              ) : null}
              <div className="u-marginTop--12 flex flex1">
                <span className="slugArrow flex justifyContent--center alignItems--center">
                  {" "}
                  {selectedRestoreApp?.slug}{" "}
                </span>
                <input
                  type="text"
                  className="Input u-position--relative"
                  style={{ textIndent: "200px" }}
                  placeholder="type your slug"
                  value={appSlugToRestore}
                  onChange={(e) => {
                    handleApplicationSlugChange(e);
                  }}
                />
              </div>
            </div>
          ) : (
            <div className="flex flex-column u-marginTop--20">
              <p className="u-fontSize--large u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--more">
                {" "}
                Which Application would you like to restore?{" "}
              </p>
              <p className="u-fontSize--normal u-fontWeight--normal u-textColor--bodyCopy u-lineHeight--normal">
                {" "}
                You can only restore one application at a time.{" "}
              </p>
              <div className="flex u-marginTop--12 u-marginBottom--30">
                <Select
                  className="replicated-select-container app-100"
                  classNamePrefix="replicated-select"
                  options={includedApps}
                  getOptionLabel={getLabel}
                  getOptionValue={(app) => app.name}
                  value={selectedRestoreApp}
                  onChange={onChangeRestoreApp}
                  isOptionSelected={(app) => {
                    app.slug === selectedRestoreApp.slug;
                  }}
                />
              </div>
            </div>
          )}
        </div>
        {selectedRestore === "full" || selectedRestore === "kotsadm" ? null : (
          <div className="flex justifyContent--flexStart u-marginTop--30">
            <button
              className="btn secondary blue u-marginRight--10"
              onClick={toggleRestoreModal}
            >
              Cancel
            </button>
            <button
              className="btn primary blue"
              onClick={() => {
                handlePartialRestoreSnapshot(
                  snapshotToRestore,
                  includedApps?.length === 1
                );
              }}
              disabled={restoringSnapshot}
            >
              {restoringSnapshot ? "Restoring..." : "Confirm and restore"}
            </button>
          </div>
        )}
      </div>
    </Modal>
  );
}
