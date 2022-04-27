import * as React from "react";
import { withRouter } from "react-router-dom";
import semverjs from "semver";
import { getBuildVersion, Utilities } from "@src/utilities/utilities";
import "@src/scss/components/shared/Footer.scss";
import ChangePassword from "./ChangePassword";

export class Footer extends React.Component {
  state = {
    targetKotsVersion: "",
  }

  componentDidMount() {
    this.setState({ targetKotsVersion: this.getHighestTargetKotsVersion()});
  }

  componentDidUpdate(lastProps) {
    if (this.props.appsList !== lastProps.appsList) {
      this.setState({ targetKotsVersion: this.getHighestTargetKotsVersion()});
    }
  }

  getHighestTargetKotsVersion = () => {
    if (!this.props.appsList) {
      return;
    }

    if (!semverjs.valid(getBuildVersion())) {
       return; 
    }

    try {
      let targetKotsVersions = [];
      for (let i = 0; i < this.props.appsList.length; i++) {
        const app = this.props.appsList[i];
        if (!app.targetKotsVersion) {
          continue;
        }
        targetKotsVersions.push(app.targetKotsVersion)
      }

      if (!targetKotsVersions.length) {
        return;
      }

      let maxSemver;
      for (let i = 0; i < targetKotsVersions.length; i++) {
        const version = targetKotsVersions[i];
        const semver = semverjs.coerce(version);
        if (!maxSemver) {
          maxSemver = semver;
          continue;
        }
        if (semverjs.gt(semver, maxSemver)) {
          maxSemver = semver;
        }
      }

      const buildSemver = semverjs.coerce(getBuildVersion());
      if (semverjs.lte(maxSemver, buildSemver)) {
        return;
      }

      return maxSemver?.version;
    } catch(err) {
      console.log(err);
    }
  }

  render() {
    return (
      <div className={`FooterContent-wrapper flex flex-auto justifyContent--center ${this.state.targetKotsVersion && "u-padding--5"} ${this.props.className || ""}`}>
        <div className="container flex1 flex">
          <div className="flex flex1 justifyContent--center alignItems--center">
            <div className="FooterItem-wrapper">
              <span className="FooterItem">{getBuildVersion()}</span>
            </div>
            {this.state.targetKotsVersion &&
              <div className="TargetKotsVersionWrapper flex u-marginLeft--10">
                <span className="icon megaPhoneIcon flex u-marginRight--10" />
                <p className="u-fontSize--small u-fontWeight--bold"> v{this.state.targetKotsVersion} available. </p>
              </div>
            }
          </div>
          {Utilities.isLoggedIn() && <ChangePassword />}
        </div>
      </div>
    );
  }
}

export default withRouter(Footer);
