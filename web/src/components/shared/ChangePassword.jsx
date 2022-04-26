import React from "react";
import ChangePasswordModal from "../modals/ChangePasswordModal/ChangePasswordModal";


const ChangePassword = () => {
  const [isOpen, setIsOpen] = React.useState(true);

  const closeModal = () => {
    setIsOpen(false);
  }

  return (
    <>
      <h1 onClick={() => setIsOpen(true)}>Change Password</h1>
      <ChangePasswordModal isOpen={isOpen} closeModal={closeModal} />
    </>
  );
}

export default ChangePassword;
