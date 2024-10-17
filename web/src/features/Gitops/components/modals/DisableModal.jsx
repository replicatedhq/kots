import Modal from "react-modal";

const DisableModal = ({ isOpen, setOpen, disableGitOps, provider }) => {
  const renderIcons = (service) => {
    if (service) {
      return <div className={`icon ${service}-icon u-marginBottom--20`} />;
    } else {
      return;
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onRequestClose={() => {
        setOpen(false);
      }}
      contentLabel="Disable GitOps"
      ariaHideApp={false}
      className="Modal"
    >
      <div className="Modal-body" style={{ width: "500px" }}>
        <div className="u-marginTop--10 u-marginBottom--10 flex flex-column alignItems--center">
          {renderIcons(provider)}
          <p className="u-fontSize--largest u-fontWeight--medium u-textColor--primary u-marginBottom--15 u-textAlign--center">
            Are you sure you want to disable GitOps <br />
            for this application?
          </p>
          <p className="u-fontSize--normal u-textColor--bodyCopy  u-textAlign--center u-lineHeight--normal">
            Commits will no longer be made to your repository, and you will have
            to <br />
            deploy from the Admin Console.
          </p>
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
              onClick={disableGitOps}
            >
              Disable GitOps
            </button>
          </div>
        </div>
      </div>
    </Modal>
  );
};

export default DisableModal;
