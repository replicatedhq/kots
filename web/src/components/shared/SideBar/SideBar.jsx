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
    items: PropTypes.array.isRequired,

    /** @type {Boolean} if true, only show loading for initial mount, and not subsequent loads */
    aggressive: PropTypes.bool
  }

  static defaultProps = {
    items: [],
    aggressive: false
  }

  shouldComponentUpdate(nextProps) {
    const { loading } = nextProps;
    const { aggressive } = this.props;

    // Don't show a loader if there is a refetch and
    // the component is set to aggressive.
    // Note this method only runs on *update*, and not for
    // the initial mount.
    if (loading && aggressive) {
      return false;
    }

    return true;
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

    if (items.length < 2) {
      return null;
    }

    return (
      <div className={classNames("sidebar u-minHeight--full", className)}>
        <div className="flex-column u-width--full u-overflow--auto">
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
