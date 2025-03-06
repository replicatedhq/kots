import { useNavigate } from "react-router-dom";
import Icon from "../Icon";
import ChangePasswordModal from "../modals/ChangePasswordModal/ChangePasswordModal";
import { useEffect, useRef, useState } from "react";

const NavBarDropdown = ({ handleLogOut, isEmbeddedCluster }) => {
  const [showDropdown, setShowDropdown] = useState(false);
  const [showModal, setShowModal] = useState(false);
  const testRef = useRef(null);
  const navigate = useNavigate();
  const closeModal = () => {
    setShowModal(false);
  };

  const handleBlur = (e) => {
    // if the clicked item is inside the dropdown
    if (e.currentTarget.contains(e.relatedTarget)) {
      // do nothing
      return;
    }
    setShowDropdown(false);
  };

  const handleNav = (e) => {
    // manually triggers nav because blur event happens too fast otherwise
    navigate("/upload-license");
    setShowDropdown(false);
  };

  useEffect(() => {
    // focus the dropdown when open so when clicked outside,
    // the onBlur event triggers and closes the dropdown
    if (showDropdown) {
      testRef.current.focus();
    }
  }, [showDropdown]);

  return (
    <div
      className="navbar-dropdown-container"
      data-testid="navbar-dropdown-container"
    >
      <span
        tabIndex={0}
        onClick={() => setShowDropdown(!showDropdown)}
        data-testid="navbar-dropdown-button"
      >
        <Icon icon="more-circle-outline" size={20} className="gray-color" />
      </span>
      <ul
        ref={testRef}
        tabIndex={0}
        onBlur={handleBlur}
        className={`dropdown-nav-menu ${showDropdown ? "" : "hidden"}`}
      >
        <li>
          <p onClick={() => setShowModal(true)}>Change password</p>
        </li>
        {!isEmbeddedCluster && (
          <li onMouseDown={handleNav} data-testid="add-new-application">
            <p>Add new application</p>
          </li>
        )}
        <li>
          <p
            data-qa="Navbar--logOutButton"
            onClick={handleLogOut}
            data-testid="log-out"
          >
            Log out
          </p>
        </li>
      </ul>
      <ChangePasswordModal isOpen={showModal} closeModal={closeModal} />
    </div>
  );
};

export default NavBarDropdown;
