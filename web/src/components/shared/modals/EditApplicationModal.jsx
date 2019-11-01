import * as React from "react";
import Modal from "react-modal";


class EditApplicationModal extends React.Component {
  render() {
    const {
      showEditModal,
      toggleEditModal,
      updateWatchInfo,
      appName,
      onFormChange,
      iconUri,
      editWatchLoading
    } = this.props;
		
    return (
      <Modal
      isOpen={showEditModal}
      onRequestClose={toggleEditModal}
      contentLabel="Yes"
      ariaHideApp={false}
      className="Modal SmallSize EditWatchModal">
      <div className="Modal-body flex-column flex1">
        <h2 className="u-fontSize--largest u-fontWeight--bold u-color--tuna u-marginBottom--10">Edit Application</h2>
        <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">You can edit the name and icon of your application</p>
        <h3 className="u-fontSize--normal u-fontWeight--bold u-color--tuna u-marginBottom--10">Application Name</h3>
        <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">This name will be shown throughout this dashboard.</p>
        <form className="EditWatchForm flex-column" onSubmit={updateWatchInfo}>
          <input
            type="text"
            className="Input u-marginBottom--20"
            placeholder="Type the app name here"
            value={appName}
            name="appName"
            onChange={onFormChange}
          />
          <h3 className="u-fontSize--normal u-fontWeight--bold u-color--tuna u-marginBottom--10">Application Icon</h3>
          <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">Provide a link to a URI to use as your app icon.</p>
          <input
            type="text"
            className="Input u-marginBotton--20"
            placeholder="Enter the link here"
            value={iconUri}
            name="iconUri"
            onChange={onFormChange}
          />
          <div className="flex justifyContent--flexEnd u-marginTop--20">
            <button
              type="button"
              onClick={toggleEditModal}
              className="btn secondary force-gray u-marginRight--20">
              Cancel
          </button>
            <button
              type="submit"
              className="btn secondary green">
              {
                editWatchLoading
                  ? "Saving"
                  : "Save Application Details"
              }
            </button>
          </div>
        </form>
      </div>
    </Modal>
    );
  }
}

export default EditApplicationModal;