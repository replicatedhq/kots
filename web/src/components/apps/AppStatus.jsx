import React from "react";
import { Link } from "react-router-dom";
import Select from "react-select";
import isEmpty from "lodash/isEmpty";
import url from "url";
import { Utilities } from "@src/utilities/utilities";

export default class AppStatus extends React.Component {

  state = {
    selectedAction: "",
  }

  componentDidMount() {
    if (this.props.links && this.props.links.length > 0) {
      this.setState({ selectedAction: this.props.links[0] })
    }
  }

  componentDidUpdate(lastProps) {
    if (this.props.links !== lastProps.links && this.props.links && this.props.links.length > 0) {
      this.setState({ selectedAction: this.props.links[0] })
    }
  }

  onActionChange = (selectedOption) => {
    if (selectedOption.uri) {
      window.open(this.createDashboardActionLink(selectedOption.uri), "_blank");
    }
    this.setState({ selectedAction: selectedOption });
  }

  createDashboardActionLink = (uri) => {
    const parsedUrl = url.parse(uri);
    let port;
    if (!parsedUrl.port) {
      port = "";
    } else {
      port = ":" + parsedUrl.port;
    }

    return `${parsedUrl.protocol}//${window.location.hostname}${port}${parsedUrl.path}`;
  }  

  render() {
    const { appStatus, url, links, app } = this.props;
    const { selectedAction } = this.state;
    return (
      <div className="flex flex1 u-marginTop--10">
      {!isEmpty(appStatus) ?
        <div className="flex alignItems--center">
          <span className={`status-dot ${appStatus === "ready" ? "u-color--success" : appStatus === "degraded" ? "u-color--warning" : "u-color--error"}`}/>
          <span className={`u-fontSize--normal u-fontWeight--medium ${appStatus === "ready" ? "u-textColor--bodyCopy" : appStatus === "degraded" ? "u-textColor--warning" : "u-textColor--error"}`}>
            {Utilities.toTitleCase(appStatus)}
          </span>
          {appStatus !== "ready" ?
            <Link to={`${url}/troubleshoot`} className="card-link u-marginLeft--10 u-borderLeft--gray u-paddingLeft--10"> Troubleshoot </Link>
            : null}
          <Link to={`${url}/config/${app?.downstreams[0]?.currentVersion?.sequence}`} className="card-link u-marginLeft--10 u-borderLeft--gray u-paddingLeft--10">Edit config</Link>
        </div>
        :
        <div className="flex alignItems--center">
          <span className="status-dot u-color--bodyCopy"/>
          <span className="u-fontSize--normal u-fontWeight--medium u-textColor--bodyCopy">
            Unknown
          </span>
        </div>
      }
      {links?.length > 0 ? // TODO: fix this and make it an inline dropdown
        <div className="flex alignItems--center u-marginLeft--10 u-borderLeft--gray u-paddingLeft--10">
          {links?.length > 1 ?
              <Select
                className="replicated-select-container"
                classNamePrefix="replicated-select"
                options={links}
                getOptionLabel={(link) => link.title}
                getOptionValue={(option) => option.title}
                value={selectedAction}
                onChange={this.onActionChange}
                isOptionSelected={(option) => { option.title === selectedAction.title }}
              />
            :
            selectedAction?.uri ?
              <a href={this.createDashboardActionLink(selectedAction.uri)} target="_blank" rel="noopener noreferrer" className="card-link"> {selectedAction.title} </a> : null
          }
        </div>
        : null
      }
    </div>
    );
  }
}
