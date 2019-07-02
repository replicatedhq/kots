import React, { Component, Fragment } from "react";
import classNames from "classnames";
import PropTypes from "prop-types";

import Loader from "@src/components/shared/Loader";
import "@src/scss/components/shared/SideBar.scss";

class SideBar extends Component {
  static propTypes = {
    /** @type {String} className to use for styling */
    className: PropTypes.string,

    /** @type {Array<JSX>} array of JSX elements to render */
    items: PropTypes.array.isRequired

  }

  static defaultProps = {
    items: [],
  }

  render() {
    const { className, items, loading } = this.props;

    if (loading) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center u-minHeight--full sidebar">
          <Loader size="60" />
        </div>
      );
    }

    return (
      <div className={classNames("sidebar flex-column flex1 u-overflow--auto", className)}>
        <div className="flex-column u-width--full">
          {items?.map((jsx, idx) => {
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
}

export default SideBar;
