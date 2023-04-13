import React from "react";
import Modal from "react-modal";
import Warning from "../../shared/Warning";
import ChangePasswordForm from "./ChangePasswordForm";
import { Link } from "react-router-dom";

import "@src/scss/components/modals/ChangePasswordModal/ChangePasswordModal.scss";
import Icon from "@src/components/Icon";

const ChangePasswordModal = ({ closeModal, isOpen }) => {
  const [passwordChangeSuccessful, setPasswordChangeSuccessful] =
    React.useState(false);
  const [identityServiceEnabled, setIdentityServiceEnabled] =
    React.useState(false);

  const handleClose = () => {
    closeModal();
    setPasswordChangeSuccessful(false);
  };

  const handleSetPasswordChangeSuccessful = (val) =>
    setPasswordChangeSuccessful(val);

  React.useEffect(() => {
    const getLoginInfo = async () => {
      try {
        const response = await fetch(`${process.env.API_ENDPOINT}/login/info`, {
          headers: {
            "Content-Type": "application/json",
          },
          method: "GET",
          credentials: "include",
        });

        if (!response.ok) {
          const res = await response.json();
          if (res.error) {
            throw new Error(
              `Unexpected status code ${response.status}: ${res.error}`
            );
          }
          throw new Error(`Unexpected status code ${response.status}`);
        }

        const loginInfo = await response.json();
        if (loginInfo?.method === "identity-service") {
          return setIdentityServiceEnabled(true);
        }
        setIdentityServiceEnabled(false);
      } catch (err) {
        console.log(err);
      }
    };
    getLoginInfo();
  }, []);

  const identityServiceContent = (
    <>
      <h3 className="u-marginTop--20">Unable to change password</h3>
      <p className="modal-text">
        Your session is currently authenticated using an identify provider and
        must be changed through that identify provider.
      </p>
      <button className="btn primary u-marginBottom--20" onClick={handleClose}>
        OK
      </button>
    </>
  );

  const standardContent = (
    <>
      {!passwordChangeSuccessful && (
        <>
          <h3>Change Admin Console Password</h3>
          <Warning>
            Changing the password for the Admin Console will invalidate and log
            out of all current sessions. Proceed with caution.
          </Warning>
          <ChangePasswordForm
            handleClose={handleClose}
            handleSetPasswordChangeSuccessful={
              handleSetPasswordChangeSuccessful
            }
          />
        </>
      )}
      {passwordChangeSuccessful && (
        <>
          <Icon
            icon="check-circle-filled"
            size={50}
            className="success-color u-marginTop--20"
          />
          <h3>Your password has been changed</h3>
          <p className="modal-text">
            Password changed successfully. You will be redirected to log in
            again. Alternatively, click below to log in.
          </p>
          <Link
            to="/secure-console"
            className="btn primary u-marginBottom--20"
            onClick={handleClose}
          >
            Log in
          </Link>
        </>
      )}
    </>
  );

  return (
    <Modal
      isOpen={isOpen}
      onRequestClose={handleClose}
      contentLabel="Change password"
      ariaHideApp={false}
      className="Modal MediumSize ChangePasswordModal"
    >
      <div
        className={`Modal-body flex-column ${
          (passwordChangeSuccessful || identityServiceEnabled) &&
          "alignItems--center"
        }`}
      >
        {identityServiceEnabled ? identityServiceContent : standardContent}
      </div>
    </Modal>
  );
};

export default ChangePasswordModal;
