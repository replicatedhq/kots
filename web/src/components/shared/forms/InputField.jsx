import React from "react";

import "@src/scss/components/shared/forms/InputField.scss";

const InputField = ({
  label,
  placeholder,
  id,
  type,
  value,
  onChange,
  autoFocus,
  helperText,
}) => {
  const [show, setShow] = React.useState(false);

  const handleToggleShow = () => {
    setShow(!show);
  };

  const calculateType = () => {
    if (type === "password") {
      return show ? "text" : "password";
    }
    return type;
  };

  const component = (
    <>
      <label className={`${id}-label`} htmlFor={id}>
        {label}
      </label>
      <p
        className="u-fontWeight--medium u-lineHeight--medium"
        style={{ width: "90%" }}
      >
        {helperText}
      </p>
      <div className="u-position--relative">
        <input
          autoFocus={!!autoFocus}
          className="Input"
          type={calculateType()}
          id={id}
          placeholder={placeholder}
          value={value}
          onChange={(e) => onChange(e)}
        />
        {type === "password" && (
          <span className="show-password-toggle" onClick={handleToggleShow}>
            {show ? "hide" : "show"}
          </span>
        )}
      </div>
    </>
  );

  return (
    <>
      {type === "password" ? (
        <div className="password-input-wrapper flex-column">{component}</div>
      ) : (
        component
      )}
    </>
  );
};

export default InputField;
