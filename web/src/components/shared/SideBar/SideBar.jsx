import React, { Fragment } from "react";
import classNames from "classnames";
import PropTypes from "prop-types";

import Loader from "@src/components/shared/Loader";
import "@src/scss/components/shared/SideBar.scss";

function SideBar(props) {
  const { className, items, loading } = props;
  if (loading) {
    return (
      <div className="flex-column flex1 alignItems--center justifyContent--center">
        <Loader size="60" />
      </div>
    );
  }

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
  items: PropTypes.array.isRequired
};

SideBar.defaultProps = {
  items: []
};

export default SideBar;
