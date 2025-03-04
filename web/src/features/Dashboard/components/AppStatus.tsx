import { Component } from "react";
import { Link } from "react-router-dom";
import InlineDropdown from "@src/components/shared/InlineDropdown";
import isEmpty from "lodash/isEmpty";
import { Utilities } from "@src/utilities/utilities";

import { App, AppStatusState } from "@types";

type OptionLink = {
  displayText: string;
  href: string;
};

type PropLink = {
  title: string;
  uri: string;
};

type Props = {
  app: App;
  appStatus?: AppStatusState | string;
  hasStatusInformers?: boolean;
  links: PropLink[];
  onViewAppStatusDetails: () => void;
  embeddedClusterState: string;
};

type State = {
  dropdownOptions: OptionLink[];
};
export default class AppStatus extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = {
      dropdownOptions: [],
    };
  }

  componentDidMount() {
    if (this.props.links && this.props.links.length > 0) {
      this.getOptions();
    }
  }

  componentDidUpdate(lastProps: Props) {
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
    let dropdownLinks: OptionLink[] = [];

    links.map((link) => {
      const linkObj = {
        displayText: link.title,
        href: this.createDashboardActionLink(link.uri),
      };
      dropdownLinks.push(linkObj);
    });
    this.setState({ dropdownOptions: dropdownLinks });
  };

  createDashboardActionLink = (uri: string) => {
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
    const { appStatus, links, app, embeddedClusterState } = this.props;
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
              data-testid="app-status-status"
            >
              {Utilities.toTitleCase(appStatus)}
            </span>
            {this.props.hasStatusInformers && (
              <span
                onClick={this.props.onViewAppStatusDetails}
                className="link u-marginLeft--10 u-fontSize--small"
              >
                {" "}
                Details{" "}
              </span>
            )}
            {!isEmpty(embeddedClusterState) && (
              <>
                <span className="tw-mr-1 tw-ml-4 tw-text-sm tw-text-gray-500">
                  Cluster State:
                </span>
                <span
                  className={`status-dot ${
                    embeddedClusterState === "Installed"
                      ? "u-color--success"
                      : embeddedClusterState === "Installing" ||
                        embeddedClusterState === "Enqueued"
                      ? "u-color--warning"
                      : "u-color--error"
                  }`}
                />
                <span
                  className={`u-fontSize--normal u-fontWeight--medium ${
                    embeddedClusterState === "Installed"
                      ? "u-textColor--bodyCopy"
                      : embeddedClusterState === "Installing" ||
                        embeddedClusterState === "Enqueued"
                      ? "u-textColor--warning"
                      : "u-textColor--error"
                  }`}
                >
                  {Utilities.humanReadableClusterState(embeddedClusterState)}
                </span>
              </>
            )}
            <Link
              to={`config/${app?.downstream?.currentVersion?.sequence}`}
              className="link u-marginLeft--10 u-borderLeft--gray u-paddingLeft--10 u-fontSize--small"
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
                className="link"
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
