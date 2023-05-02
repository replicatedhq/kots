import React from "react";

import "@src/scss/components/shared/forms/InputField.scss";
import Icon from "@components/Icon";
import { usePrevious } from "@src/hooks/usePrevious";

const InputField = ({
  label,
  placeholder,
  id,
  type,
  value,
  onChange,
  onFocus,
  onBlur,
  className,
  autoFocus,
  helperText,
  isFirstChange,
  showError = false,
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

  const prevValue = usePrevious(value);

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
          className={`Input ${showError ? "has-error" : ""}`}
          type={calculateType()}
          id={id}
          placeholder={placeholder}
          value={value}
          onChange={(e) => onChange(e)}
          onBlur={onBlur}
          onFocus={onFocus}
        />
        {type !== "password" && showError && (
          <span className="show-input-error">
            <Icon
              icon={"warning-circle-filled"}
              size={16}
              className="error-color"
            />
          </span>
        )}
        {type === "password" && prevValue === "" && (
          <span className="show-password-toggle" onClick={handleToggleShow}>
            {
              <Icon
                icon={show ? "visible" : "visibility-off"}
                size={16}
                className="gray-color"
              />
            }
          </span>
        )}
      </div>
    </>
  );

  return (
    <>
      {type === "password" ? (
        <div className={`password-input-wrapper flex-column ${className} `}>
          {component}
        </div>
      ) : (
        component
      )}
    </>
  );
};

export default InputField;
