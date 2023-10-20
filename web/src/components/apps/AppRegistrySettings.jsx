import { Component } from "react";
import { KotsPageTitle } from "@components/Head";
import AirgapRegistrySettings from "../shared/AirgapRegistrySettings";

export default class AppRegistrySettings extends Component {
  render() {
    const { app, updateCallback } = this.props;

    return (
      <div className="flex justifyContent--center">
        <KotsPageTitle pageName="Registry Settings" showAppSlug />
        <div className="AirgapSettings--wrapper card-bg u-textAlign--left u-marginTop--30 u-paddingRight--20 u-paddingLeft--20">
          <p className="u-fontWeight--bold card-title u-fontSize--large u-marginBottom--10 u-lineHeight--normal">
            Registry settings
          </p>
          <AirgapRegistrySettings app={app} updateCallback={updateCallback} />
        </div>
      </div>
    );
  }
}
