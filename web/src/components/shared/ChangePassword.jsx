import React from "react";
import ChangePasswordModal from "../modals/ChangePasswordModal/ChangePasswordModal";


const ChangePassword = () => {
  const [isOpen, setIsOpen] = React.useState(false);

  const closeModal = () => {
    setIsOpen(false);
  }

  return (
    <>
      <h1
        className="FooterItem u-textDecoration--underline u-cursor--pointer"
        onClick={() => setIsOpen(true)}
      >
        Change Password
      </h1>
      <ChangePasswordModal isOpen={isOpen} closeModal={closeModal} />
    </>
  );
}

export default ChangePassword;
