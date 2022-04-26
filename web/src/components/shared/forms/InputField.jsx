import React from "react";

const InputField = ({
  label,
  placeholder,
  id,
  type,
  value,
  onChange,
}) => {
  return (
    <>
      <label className={`${id}-label`} htmlFor={id}>{label}</label>
      <input
        className="Input"
        type={type}
        id={id}
        placeholder={placeholder}
        value={value}
        onChange={(e) => onChange(e)}
      />
    </>
  )
}

export default InputField;
