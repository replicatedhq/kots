import Modal from "react-modal";
import Icon from "../Icon";

const AddANodeModal = ({
  displayAddNode,
  toggleDisplayAddNode,
  rolesData,
  children,
}) => {
  return (
    <Modal
      isOpen={displayAddNode}
      onRequestClose={() => toggleDisplayAddNode()}
      contentLabel="Add Node"
      className="Modal"
      ariaHideApp={false}
    >
      <div className="Modal-body tw-flex tw-flex-col tw-gap-4 tw-font-sans">
        <div className="tw-flex">
          <h1 className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--more">
            Add a Node
          </h1>
          <Icon
            icon="close"
            size={14}
            className="tw-ml-auto gray-color clickable close-icon"
            onClick={() => toggleDisplayAddNode()}
          />
        </div>
        <p className="tw-text-base tw-text-gray-600">
          {rolesData?.roles &&
            rolesData.roles.length > 1 &&
            "Select one or more roles to assign to the new node. "}
          Copy the join command and run it on the machine you'd like to join to
          the cluster.
        </p>
        {children}
        <div className="tw-w-full tw-flex tw-justify-end tw-gap-2">
          <button
            className="btn secondary large"
            onClick={() => toggleDisplayAddNode()}
          >
            Close
          </button>
        </div>
      </div>
    </Modal>
  );
};

export default AddANodeModal;
