import React from "react";
import classNames from "classnames";

export default function SidebarLayout(props) {
  const {
    className,
    children,
    sidebar,
    sidebarProps = {},
    condition = true
  } = props;

  return (
    <div className={classNames(className)}>
      {condition && (
        <div className="flex">
          {sidebar}
        </div>
      )}
      {children}
    </div>
  );
}
