import React from "react";
import Modal from "react-modal";
import Loader from "../../shared/Loader";
import { useHistory } from "react-router";

const ConnectionModal = ({
  isOpen,
  modalType,
  setOpen,
  handleTestConnection,
  isTestingConnection,
  stepFrom,
  appSlug,
  getAppsList,
  getGitops,
}) => {
  const history = useHistory();
  switch (modalType) {
    case "success":
      return (
        <Modal
          isOpen={isOpen}
          onRequestClose={() => {
            setOpen(false);
          }}
          contentLabel="Connection to repository"
          ariaHideApp={false}
          className="Modal"
        >
          <div className="Modal-body" style={{ width: "500px" }}>
            <div className="u-marginTop--10 u-marginBottom--10 flex flex-column alignItems--center">
              <div
                className={
                  "icon success-checkmark-icon-bright u-marginBottom--20"
                }
              ></div>
              <p className="u-fontSize--largest u-fontWeight--medium u-textColor--primary u-marginBottom--15">
                GitOps is enabled
              </p>
              <p className="u-fontSize--normal u-textColor--bodyCopy  u-textAlign--center u-lineHeight--normal">
                Updates will be committed to your repository to be deployed.
              </p>
              <div className="u-marginTop--30">
                <button
                  type="button"
                  className="btn secondary blue u-marginRight--10"
                  onClick={async () => {
                    await getAppsList();
                    await getGitops();
                    setOpen(false);
                    stepFrom("action", "provider");
                  }}
                >
                  View configuration
                </button>
                <button
                  type="button"
                  className="btn primary blue"
                  //TODO: WORK ON THIS
                  onClick={() => history.push(`/app/${appSlug}`)}
                >
                  Go to dashboard
                </button>
              </div>
            </div>
          </div>
        </Modal>
      );
    case "fail":
      return (
        <Modal
          isOpen={isOpen}
          onRequestClose={() => {
            setOpen(false);
          }}
          contentLabel="Connection to repository"
          ariaHideApp={false}
          className="Modal"
        >
          <div className="Modal-body" style={{ width: "500px" }}>
            <div className="u-marginTop--10 u-marginBottom--10 flex flex-column alignItems--center">
              <div
                className={
                  "icon instanceSnapshot-warning-icon-large u-marginBottom--20"
                }
              ></div>
              <p className="u-fontSize--largest u-fontWeight--medium u-textColor--primary u-marginBottom--15">
                Connection to repository failed
              </p>
              <p className="u-fontSize--normal u-textColor--bodyCopy u-textAlign--center u-lineHeight--normal">
                Ensure that you added the SSH key with write access, and that
                the repository has at least one file committed to it already.
              </p>
              {isTestingConnection ? (
                <div className="u-marginTop--30">
                  <Loader size="30" />
                </div>
              ) : (
                <div className="u-marginTop--30">
                  <button
                    type="button"
                    className="btn secondary blue u-marginRight--10"
                    onClick={() => {
                      setOpen(false);
                    }}
                  >
                    Cancel
                  </button>
                  <button
                    type="button"
                    className="btn primary blue"
                    onClick={handleTestConnection}
                  >
                    Try again
                  </button>
                </div>
              )}
            </div>
          </div>
        </Modal>
      );
    default:
      return <div></div>;
  }
};

export default ConnectionModal;
