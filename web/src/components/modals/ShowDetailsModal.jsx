import Modal from "react-modal";
import { Link } from "react-router-dom";

export default function ShowDetailsModal(props) {
  const {
    displayShowDetailsModal,
    toggleShowDetailsModal,
    yamlErrorDetails,
    deployView,
    showDeployWarningModal,
    showSkipModal,
    forceDeploy,
    slug,
    sequence,
  } = props;

  return (
    <Modal
      isOpen={displayShowDetailsModal}
      shouldReturnFocusAfterClose={false}
      onRequestClose={() => {
        toggleShowDetailsModal({});
      }}
      ariaHideApp={false}
      contentLabel="Modal"
      className="Modal MediumSize"
    >
      <div className="Modal-body">
        <div className="flex flex-column">
          <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--more">
            {" "}
            Invalid files in your application{" "}
          </p>
          <p className="u-fontSize--normal u-fontWeight--normal u-textColor--bodyCopy u-lineHeight--normal">
            Your application can be deployed, but the files with the errors will
            not be included.{" "}
          </p>

          <div className="u-marginTop--20">
            <p className="u-fontSize--large u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-borderBottom--gray darker u-paddingBottom--10">
              {" "}
              The following files contain errors{" "}
            </p>
            {yamlErrorDetails?.map((err, i) => (
              <div
                className="flex flex1 alignItems--center u-borderBottom--gray darker u-paddingTop--10 u-paddingBottom--10"
                key={i}
              >
                <div className="flex">
                  <span className="icon invalid-yaml-icon" />
                </div>
                <div className="flex flex-column u-marginLeft--10">
                  <div className="flex flex1 alignItems--center">
                    <span className="u-fontSize--large u-fontWeight--bold u-textColor--primary u-lineHeight--normal">
                      {" "}
                      {err.path}{" "}
                    </span>
                    <Link
                      to={`/app/${slug}/tree/${sequence}?${err.path}`}
                      className="link u-marginLeft--5 u-marginTop--5 u-fontSize--small"
                    >
                      {" "}
                      View{" "}
                    </Link>
                  </div>
                  <span className="u-fontSize--small u-fontWeight--medium u-textColor--error u-lineHeight--normal">
                    {" "}
                    error: {err.error}{" "}
                  </span>
                </div>
              </div>
            ))}
          </div>
          {deployView ? (
            <div className="flex justifyContent--flexStart u-marginTop--20">
              <button
                className="btn primary blue"
                onClick={() => {
                  showDeployWarningModal || showSkipModal
                    ? toggleShowDetailsModal()
                    : forceDeploy();
                }}
              >
                Deploy
              </button>
              <button
                className="btn secondary u-marginLeft--20"
                onClick={() => {
                  toggleShowDetailsModal();
                }}
              >
                Cancel
              </button>
            </div>
          ) : (
            <div className="flex justifyContent--flexStart u-marginTop--20">
              <button
                className="btn primary blue"
                onClick={() => {
                  toggleShowDetailsModal();
                }}
              >
                Ok, got it!
              </button>
            </div>
          )}
        </div>
      </div>
    </Modal>
  );
}
