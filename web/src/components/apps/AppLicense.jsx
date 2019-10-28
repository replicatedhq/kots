import React, { Component } from "react";
import Helmet from "react-helmet";

import {
  getLicenseExpiryDate,
} from "@src/utilities/utilities";

import { graphql, compose, withApollo } from "react-apollo";
import { getAppLicense } from "@src/queries/AppsQueries";
import Loader from "../shared/Loader";

class AppLicense extends Component {

  constructor(props) {
    super(props);

    this.state = {
      appLicense: null
    }
  }

  componentDidMount() {
    const { getAppLicense } = this.props.getAppLicense;
    if (getAppLicense) {
      this.setState({ appLicense: getAppLicense });
    }
  }

  componentDidUpdate(lastProps) {
    if (this.props.getAppLicense !== lastProps.getAppLicense && this.props.getAppLicense) {
      const { getAppLicense } = this.props.getAppLicense;
      if (getAppLicense) {
        this.setState({ appLicense: getAppLicense });
      }
    }
  }

  render() {
    const { appLicense } = this.state;

    if (!appLicense) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      );
    }

    const { app } = this.props;

    const expiresAt = getLicenseExpiryDate(appLicense);

    return (
      <div className="flex justifyContent--center">
        <Helmet>
          <title>{`${app.name} License`}</title>
        </Helmet>
        <div className="LicenseDetails--wrapper u-textAlign--left u-paddingRight--20 u-paddingLeft--20">
          <div className="flex u-marginBottom--20 u-paddingBottom--5 u-marginTop--20 alignItems--center">
            <p className="u-fontWeight--bold u-color--tuna u-fontSize--larger u-lineHeight--normal u-marginRight--10">License details</p>
          </div>
          <div className="u-color--tundora u-fontSize--normal u-fontWeight--medium">
            <div className="flex u-marginBottom--20">
              <p className="u-marginRight--10">Expires:</p>
              <p className="u-fontWeight--bold u-color--tuna">{expiresAt}</p>
            </div>
            {appLicense.channelName && 
              <div className="flex u-marginBottom--20">
                <p className="u-marginRight--10">Channel:</p>
                <p className="u-fontWeight--bold u-color--tuna">{appLicense.channelName}</p>
              </div>
            }
            {appLicense.entitlements?.map(entitlement => {
              return (
                <div key={entitlement.label} className="flex u-marginBottom--20">
                  <p className="u-marginRight--10">{entitlement.title}</p>
                  <p className="u-fontWeight--bold u-color--tuna">{entitlement.value}</p>
                </div>
              );
            })}
          </div>
        </div>
      </div>
    );
  }
}

export default compose(
  withApollo,
  graphql(getAppLicense, {
    name: "getAppLicense",
    options: props => {
      return {
        variables: {
          appId: props.app.id
        },
        fetchPolicy: "no-cache"
      };
    }
  })
)(AppLicense);
