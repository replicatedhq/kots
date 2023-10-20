import { useState } from "react";
import InputField from "../../shared/forms/InputField";

import "@src/scss/components/modals/ChangePasswordModal/ChangePasswordForm.scss";

const ChangePasswordForm = ({
  handleClose,
  handleSetPasswordChangeSuccessful,
}) => {
  const [err, setErr] = useState({
    status: false,
    message: "",
  });
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    currentPassword: "",
    newPassword: "",
    confirmPassword: "",
  });

  const updateFormStatus = (loadingStatus, errStatus, message) => {
    setErr({
      status: errStatus,
      message: message,
    });
    setLoading(loadingStatus);
  };

  const validatePassword = () => {
    if (!inputs.currentPassword || inputs.currentPassword.length === "0") {
      updateFormStatus(false, true, "Current password is required.");
      return false;
    }
    if (
      !inputs.newPassword ||
      !inputs.confirmPassword ||
      inputs.newPassword.length === "0" ||
      inputs.confirmPassword.length === "0"
    ) {
      updateFormStatus(
        false,
        true,
        "Please ensure you've filled out both new password fields."
      );
      return false;
    }
    if (inputs.newPassword !== inputs.confirmPassword) {
      updateFormStatus(false, true, "Passwords do not match.");
      return false;
    }
    return true;
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    if (validatePassword()) {
      updateFormStatus(true, false, "");

      fetch(`${process.env.API_ENDPOINT}/password/change`, {
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        method: "PUT",
        body: JSON.stringify({
          current_password: inputs.currentPassword,
          new_password: inputs.newPassword,
        }),
      })
        .then(async (res) => {
          if (res.status >= 400) {
            let body = await res.json();
            let msg = body.error;
            if (!msg) {
              msg =
                res.status === 401
                  ? "User unauthorized. Please try again"
                  : "There was an error changing your password. Please try again.";
            }
            updateFormStatus(false, true, msg);
            handleSetPasswordChangeSuccessful(false);
            return;
          }
          handleSetPasswordChangeSuccessful(true);
        })
        .catch((err) => {
          console.log("Login failed:", err);
          updateFormStatus(
            false,
            true,
            "There was an error changing your password. Please try again."
          );
          handleSetPasswordChangeSuccessful(false);
        });
    }
  };

  return (
    <form
      className="change-password-form flex-column"
      onSubmit={(e) => handleSubmit(e)}
    >
      <InputField
        autoFocus
        label="Current password"
        id="current-password"
        placeholder="current password"
        type="password"
        value={inputs.currentPassword}
        onChange={(e) =>
          setInputs({ ...inputs, currentPassword: e.target.value })
        }
        isFirstChange={true}
      />
      <InputField
        label="New password"
        helperText="Password must be at least 6 characters and must not be the same as your current password."
        id="new-password"
        placeholder="new password"
        type="password"
        value={inputs.newPassword}
        onChange={(e) => setInputs({ ...inputs, newPassword: e.target.value })}
        isFirstChange={true}
      />
      <InputField
        label="Confirm new password"
        id="new-password-confirm"
        placeholder="confirm new password"
        value={inputs.confirmPassword}
        type="password"
        onChange={(e) =>
          setInputs({ ...inputs, confirmPassword: e.target.value })
        }
        isFirstChange={true}
      />
      <div className="flex change-password-submit-section">
        <button
          type="reset"
          className="btn secondary blue"
          onClick={handleClose}
        >
          Cancel
        </button>
        <button className="btn primary" type="submit">
          Change Password
        </button>
        {err.status && <p className="change-password-error">{err.message}</p>}
      </div>
    </form>
  );
};

export default ChangePasswordForm;
