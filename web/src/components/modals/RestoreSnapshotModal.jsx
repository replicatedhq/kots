import Modal from "react-modal";
import { Utilities } from "../../utilities/utilities";

export default function RestoreSnapshotModal(props) {
  const {
    restoreSnapshotModal,
    toggleRestoreModal,
    snapshotToRestore,
    handleRestoreSnapshot,
    restoringSnapshot,
    restoreErr,
    restoreErrorMsg,
    app,
    appSlugToRestore,
    appSlugMismatch,
    handleApplicationSlugChange,
  } = props;

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
          <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--more">
            Restore from Partial backup (Application)
          </p>
          <p className="u-fontSize--normal u-fontWeight--normal u-textColor--bodyCopy u-lineHeight--normal">
            This will be a partial restore of your application and its metadata.
          </p>
          {restoreErr ? (
            <p className="u-textColor--error u-fontSize--small u-fontWeight--medium u-lineHeight--normal">
              {restoreErrorMsg}
            </p>
          ) : null}
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
              <p className="u-fontSize--normal u-textColor--bodyCopy u-fontWeight--bold u-lineHeight--normal justifyContent--center flex alignItems--center">
                <span className="icon snapshot-volume-icon" />{" "}
                {snapshotToRestore?.volumeSuccessCount}/
                {snapshotToRestore?.volumeCount}
              </p>
            </div>
          </div>
          <div className="flex flex1 u-marginTop--20">
            <p className="u-fontSize--normal u-fontWeight--normal u-textColor--bodyCopy u-lineHeight--normal">
              {" "}
              Restoring to this version will remove data and replace it with
              data from the restored version. During the restoration, your
              application will not be available and you will not be able to use
              the admin console. This action cannot be reversed.{" "}
            </p>
          </div>
          <div className="flex flex-column u-marginTop--20">
            <p className="u-fontSize--normal u-fontWeight--bold u-textColor--primary u-lineHeight--normal">
              {" "}
              Type your application slug to continue
            </p>

            <p className="u-fontSize--normal u-fontWeight--normal u-textColor--bodyCopy u-lineHeight--normal">
              To confirm that you want to restore this snapshot, please type
              it's slug in the input as it appears below.
            </p>
            {appSlugMismatch ? (
              <p className="u-textColor--error u-fontSize--small u-fontWeight--medium u-lineHeight--normal">
                The app slug you entered does not match the current app slug
              </p>
            ) : null}
            <div className="u-marginTop--12 flex flex1">
              <span className="slugArrow flex justifyContent--center alignItems--center">
                {" "}
                {app?.slug}{" "}
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
          <div className="flex justifyContent--flexStart u-marginTop--30">
            <button
              className="btn secondary blue u-marginRight--10"
              onClick={() => {
                toggleRestoreModal({});
              }}
            >
              Cancel
            </button>
            <button
              className="btn primary blue"
              onClick={() => {
                handleRestoreSnapshot(snapshotToRestore);
              }}
              disabled={restoringSnapshot}
            >
              {restoringSnapshot ? "Restoring..." : "Confirm and restore"}
            </button>
          </div>
        </div>
      </div>
    </Modal>
  );
}
