import React, { Component } from "react";
import Helmet from "react-helmet";
import AirgapRegistrySettings from "../shared/AirgapRegistrySettings";

export default class AppRegistrySettings extends Component {
  render() {
    const { app, updateCallback } = this.props;

    return (
      <div className="flex justifyContent--center">
        <Helmet>
          <title>{`${app.name} Airgap settings`}</title>
        </Helmet>
        <div className="AirgapSettings--wrapper u-textAlign--left u-marginTop--30 u-paddingRight--20 u-paddingLeft--20">
          <p className="u-fontWeight--bold u-textColor--primary u-fontSize--larger u-marginTop--15 u-marginBottom--10 u-lineHeight--normal">
            Registry settings
          </p>
          <AirgapRegistrySettings app={app} updateCallback={updateCallback} />
        </div>
      </div>
    );
  }
}
