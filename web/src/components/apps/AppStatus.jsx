import React from "react";
import { Link } from "react-router-dom";
import InlineDropdown from "../shared/InlineDropdown";
import isEmpty from "lodash/isEmpty";
import { Utilities } from "@src/utilities/utilities";

export default class AppStatus extends React.Component {
  state = {
    dropdownOptions: [],
  };

  componentDidMount() {
    if (this.props.links && this.props.links.length > 0) {
      this.getOptions();
    }
  }

  componentDidUpdate(lastProps) {
    if (
      this.props.links !== lastProps.links &&
      this.props.links &&
      this.props.links.length > 0
    ) {
      this.getOptions();
    }
  }

  getOptions = () => {
    const { links } = this.props;
    let dropdownLinks = [];

    links.map((link) => {
      const linkObj = {
        displayText: link.title,
        href: this.createDashboardActionLink(link.uri),
      };
      dropdownLinks.push(linkObj);
    });
    this.setState({ dropdownOptions: dropdownLinks });
  };

  createDashboardActionLink = (uri) => {
    try {
      const parsedUrl = new URL(uri);
      if (parsedUrl.hostname === "localhost") {
        parsedUrl.hostname = window.location.hostname;
      }
      return parsedUrl.href;
    } catch (error) {
      return "";
    }
  };

  render() {
    const { appStatus, url, links, app } = this.props;
    const { dropdownOptions } = this.state;
    const defaultDisplayText =
      dropdownOptions.length > 0 ? dropdownOptions[0].displayText : "";

    return (
      <div className="flex flex1 u-marginTop--10">
        {!isEmpty(appStatus) ? (
          <div className="flex alignItems--center">
            <span
              className={`status-dot ${
                appStatus === "ready"
                  ? "u-color--success"
                  : appStatus === "degraded" || appStatus === "updating"
                  ? "u-color--warning"
                  : "u-color--error"
              }`}
            />
            <span
              className={`u-fontSize--normal u-fontWeight--medium ${
                appStatus === "ready"
                  ? "u-textColor--bodyCopy"
                  : appStatus === "degraded" || appStatus === "updating"
                  ? "u-textColor--warning"
                  : "u-textColor--error"
              }`}
            >
              {Utilities.toTitleCase(appStatus)}
            </span>
            {this.props.hasStatusInformers && (
              <span
                onClick={this.props.onViewAppStatusDetails}
                className="card-link u-marginLeft--10"
              >
                {" "}
                Details{" "}
              </span>
            )}
            <Link
              to={`${url}/config/${app?.downstream?.currentVersion?.sequence}`}
              className="card-link u-marginLeft--10 u-borderLeft--gray u-paddingLeft--10"
            >
              Edit config
            </Link>
          </div>
        ) : (
          <div className="flex alignItems--center">
            <span className="status-dot u-color--bodyCopy" />
            <span className="u-fontSize--normal u-fontWeight--medium u-textColor--bodyCopy">
              Unknown
            </span>
          </div>
        )}
        {links?.length > 0 ? ( // TODO: fix this and make it an inline dropdown
          <div className="flex alignItems--center u-marginLeft--10 u-borderLeft--gray u-paddingLeft--10 u-fontSize--small u-fontWeight--medium">
            {links?.length > 1 ? (
              <InlineDropdown
                defaultDisplayText={defaultDisplayText}
                dropdownOptions={dropdownOptions}
              />
            ) : links[0]?.uri ? (
              <a
                href={this.createDashboardActionLink(links[0].uri)}
                target="_blank"
                rel="noopener noreferrer"
                className="card-link"
              >
                {" "}
                {links[0].title}{" "}
              </a>
            ) : null}
          </div>
        ) : null}
      </div>
    );
  }
}
