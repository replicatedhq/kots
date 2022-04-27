import React from "react";

const InputField = ({
  label,
  placeholder,
  id,
  type,
  value,
  onChange,
}) => {
  const [show, setShow] = React.useState(false);

  const handleToggleShow = () => {
    setShow(!show);
  }

  const calculateType = () => {
    if (type === "password") {
      return show ? "text" : "password";
    }
    return type;
  }

  const component = (
    <>
      <label className={`${id}-label`} htmlFor={id}>{label}</label>
      <input
        className="Input"
        type={calculateType()}
        id={id}
        placeholder={placeholder}
        value={value}
        onChange={(e) => onChange(e)}
      />
      {type === "password" &&
        <span className="show-password-toggle" onClick={handleToggleShow}>
          {show ? "hide" : "show"}
        </span>
      }
    </>
  )

  return (
    <>
      {type === "password" ? (
        <div className="password-input-wrapper flex-column">
          {component}
        </div>
      ) : component}
    </>
   )
}

export default InputField;
