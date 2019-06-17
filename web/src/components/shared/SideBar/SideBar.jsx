import React, { Fragment } from "react";
import classNames from "classnames";
import PropTypes from "prop-types";
import { Link } from "react-router-dom";

import "@src/scss/components/shared/SideBar.scss";

function SideBar(props) {
  const { className, items } = props;

  return (
    <div className={classNames("sidebar u-minHeight--full", className)}>
      <div className="flex-column u-width--full">
        {items?.map( (jsx, idx) => {
          return (
            <Fragment key={idx}>
              {jsx}
            </Fragment>
          );
        })}
      </div>
    </div>
  );
}

SideBar.displayName = "SideBar";

SideBar.propTypes = {
  className: PropTypes.string,
  currentWatch: PropTypes.string
};
export default SideBar;
