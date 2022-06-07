import React from "react";
import { Link } from "react-router-dom";
import ChangePasswordModal from "../modals/ChangePasswordModal/ChangePasswordModal";

const NavBarDropdown = ({ handleLogOut }) => {
  const [showDropdown, setShowDropdown] = React.useState(false);
  const [showModal, setShowModal] = React.useState(false);
  const testRef = React.useRef(null);

  const closeModal = () => {
    setShowModal(false);
  };

  React.useEffect(() => {
    // focus the dropdown when open so when clicked outside,
    // the onBlur event triggers and closes the dropdown
    if (showDropdown) {
      testRef.current.focus();
    }
  }, [showDropdown]);

  return (
    <div className="navbar-dropdown-container">
      <span
        tabIndex={0}
        onClick={() => setShowDropdown(!showDropdown)}
        className="icon menu-dots-icon"
      />
      <ul
        ref={testRef}
        tabIndex={0}
        onBlur={() => setShowDropdown(false)}
        className={`dropdown-nav-menu ${showDropdown ? "" : "hidden"}`}
      >
        <li>
          <p onClick={() => setShowModal(true)}>Change Password</p>
        </li>
        <li>
          <Link to="/upload-license">Add new application</Link>
        </li>
        <li>
          <p data-qa="Navbar--logOutButton" onClick={handleLogOut}>
            Log out
          </p>
        </li>
      </ul>
      <ChangePasswordModal isOpen={showModal} closeModal={closeModal} />
    </div>
  );
};

export default NavBarDropdown;
