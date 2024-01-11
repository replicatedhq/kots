import Modal from "react-modal";

export default function ShowAllModal(props) {
  const { displayShowAllModal, toggleShowAllModal, dataToShow, name } = props;

  return (
    <Modal
      isOpen={displayShowAllModal}
      onRequestClose={toggleShowAllModal}
      shouldReturnFocusAfterClose={false}
      contentLabel="Show more"
      ariaHideApp={false}
      className="MediumSize Modal"
    >
      <div className="Modal-body flex-column flex1">
        <p className="u-fontSize--larger u-textColor--primary u-fontWeight--bold u-lineHeight--bold u-paddingBottom--10 u-borderBottom--gray">
          {name}
        </p>
        {dataToShow}
        <div className="u-marginTop--10 flex">
          <button
            onClick={() => toggleShowAllModal()}
            className="btn primary blue"
          >
            Ok, got it!
          </button>
        </div>
      </div>
    </Modal>
  );
}
