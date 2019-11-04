import React, { Component } from "react";
import Helmet from "react-helmet";
import AppGitopsSettings from "../shared/AppGitopsSettings";

export default class AppGitops extends Component {

  render() {
    const { app } = this.props;

    return (
      <div className="flex justifyContent--center">
        <Helmet>
          <title>{`${app.name} GitOps settings`}</title>
        </Helmet>
        <div className="GitopsSettings--wrapper u-textAlign--left u-paddingRight--20 u-paddingLeft--20">
          <p className="u-fontWeight--bold u-color--tuna u-fontSize--larger u-marginTop--30 u-marginBottom--20 u-paddingBottom--5 u-lineHeight--normal">GitOps settings</p>
          <AppGitopsSettings app={app} />
        </div>
      </div>
    );
  }
}
