import React, { Component, Fragment } from "react";
import classNames from "classnames";
import PropTypes from "prop-types";
import { withRouter, Link } from "react-router-dom";

import addAppIcon from "../../../images/add-application.svg";
import consoleSettingsIcon from "../../../images/console-settings.svg";
import Loader from "@src/components/shared/Loader";
import "@src/scss/components/shared/SideBar.scss";

class SideBar extends Component {
  static propTypes = {
    /** @type {String} className to use for styling */
    className: PropTypes.string,

    /** @type {Array<JSX>} array of JSX elements to render */
    items: PropTypes.array.isRequired,

    /** @type {Function} function to toggle open state of sidebar */
    toggleSidebar: PropTypes.func.isRequired

  }

  static defaultProps = {
    items: [],
  }

  expandSidebar = () => {
    this.props.toggleSidebar(true)
  }

  closeSidebar = () => {
    this.props.toggleSidebar(false)
  }

  render() {
    const { className, items, loading } = this.props;
    const { pathname } = this.props.location;

    if (loading) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center u-minHeight--full sidebar">
          <Loader size="60" />
        </div>
      );
    }

    return (
      <div className={classNames("sidebar flex-column flex-auto u-overflow--auto", className, { "expanded": this.props.sidebarOpen })} onMouseEnter={this.expandSidebar} onMouseLeave={this.closeSidebar}>
        <div className="flex-column flex1">
          <div className="flex1">
            {items?.map((jsx, idx) => {
              return (
                <Fragment key={idx}>
                  {jsx}
                </Fragment>
              );
            })}
            <div className="sidebar-link">
              <Link
                className="flex alignItems--center"
                to="/upload-license">
                  <span className="sidebar-link-icon add-app" style={{ backgroundImage: `url(${addAppIcon})` }}></span>
                  {this.props.sidebarOpen &&
                    <div className="flex-column u-marginLeft--10">
                      <p className="u-color--tuna u-fontSize--normal u-fontWeight--bold">Add application</p>
                    </div>
                  }
              </Link>
            </div>
          </div>
          <div className="flex-auto">
            <div className={classNames("sidebar-link", { selected: pathname.startsWith("/settings") })}>
              <Link
                className="flex alignItems--center"
                to="/settings/authentication">
                  <span className="sidebar-link-icon console-settings" style={{ backgroundImage: `url(${consoleSettingsIcon})` }}></span>
                  {this.props.sidebarOpen &&
                    <div className="flex-column u-marginLeft--10">
                      <p className="u-color--tuna u-fontSize--normal u-fontWeight--bold">Console settings</p>
                    </div>
                  }
              </Link>
            </div>
          </div>
        </div>
      </div>
    );
  }
}

export default withRouter(SideBar);
