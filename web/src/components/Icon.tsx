// Icon.jsx
import React from "react";
import IcoMoon from "react-icomoon";
import iconSet from "./selection.json";
import "@src/css/icon.css";

type IconProps = {
  icon: string;
  size: number | string;
  color?: string;
  style?: object;
  className?: string;
  disableFill?: boolean;
  removeInlineStyle?: boolean;
};

const Icon = (props: IconProps) => {
  let className = props.className ? props.className : "";
  return (
    <IcoMoon iconSet={iconSet} {...props} className={`icomoon ${className}`} />
  );
};

export default Icon;
