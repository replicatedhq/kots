import React from "react";
import Modal from "react-modal";
import Warning from "../../shared/Warning";
import ChangePasswordForm from "./ChangePasswordForm";
import { Link } from "react-router-dom";

import "@src/scss/components/modals/ChangePasswordModal.scss";

const ChangePasswordModal = ({ closeModal, isOpen }) => {
  const [isSuccessful, setIsSuccessful] = React.useState(false);

  const handleClose = () => {
    closeModal();
    setIsSuccessful(false);
  }

  const handleSetIsSuccessful = (val) => setIsSuccessful(val);

  return (
    <Modal
      isOpen={isOpen}
      onRequestClose={handleClose}
      contentLabel="Change password"
      ariaHideApp={false}
      className="Modal MediumSize ChangePasswordModal"
    >
      <div className={`Modal-body flex-column ${isSuccessful && "alignItems--center"}`}>
        {!isSuccessful &&
          <>
            <h3>Change Admin Console Password</h3>
            <Warning>
              Changing the password for the Admin Console will invalidate and log out of all current sessions. Proceed with caution.
            </Warning>
            <ChangePasswordForm handleClose={handleClose} handleSetIsSuccessful={handleSetIsSuccessful} />
          </>
        }
        {isSuccessful &&
          <>
            <span className="icon success-checkmark-icon-bright u-marginTop--20" />
            <h3>Your password has been changed</h3>
            <p className="password-success-message">
              Password changed successfully. You will be redirected to log in again. Alternatively, click below to log in.
            </p>
            <Link to="/secure-console" className="btn primary u-marginBottom--20" onClick={handleClose}>Log in</Link>
          </>
        }
      </div>
    </Modal>
  );
}

export default ChangePasswordModal;
