// Icon.jsx
import IcoMoon from "react-icomoon";
import iconSet from "./selection.json";
import "@src/css/icon.css";

type IconProps = {
  icon: string;
  size: number;
  color?: string;
  style?: object;
  className?: string;
  disableFill?: boolean;
  removeInlineStyle?: boolean;
  onClick?: () => void;
};

const Icon = (props: IconProps) => {
  let className = props.className ? props.className : "";
  return (
    <IcoMoon iconSet={iconSet} {...props} className={`icons ${className}`} />
  );
};

export default Icon;
