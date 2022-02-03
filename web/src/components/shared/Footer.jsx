import * as React from "react";
import { withRouter } from "react-router-dom";
import semverjs from "semver";
import { getBuildVersion } from "../../utilities/utilities";
import "../../scss/components/shared/Footer.scss";

export class Footer extends React.Component {
  state = {
    targetKotsVersion: ""
  }

  componentDidMount() {
    this.getHighestTargetKotsVersion();
  }

  componentDidUpdate(lastProps) {
    if (this.props.appsList !== lastProps.appsList) {
      this.getHighestTargetKotsVersion();
    }
  }

  getHighestTargetKotsVersion = () => {
    if (!this.props.appsList) {
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

      this.setState({
        targetKotsVersion: maxSemver?.version
      });
    } catch(err) {
      console.log(err);
    }
  }

  render() {
    return (
      <div className={`FooterContent-wrapper flex flex-auto justifyContent--center ${this.props.className || ""}`}>
        <div className="container flex1 flex">
          <div className="flex flex1 justifyContent--center">
            <div className="FooterItem-wrapper">
              <span className="FooterItem">{getBuildVersion()}</span>
            </div>
            {this.state.targetKotsVersion &&
              <div className="flex FooterItem-wrapper u-marginLeft--10">
                <span className="icon info-warning-icon flex u-marginRight--5" />
                <p className="TargetKotsVersion u-fontSize--small u-marginRight--5"> A newer supported version of KOTS is available. Update KOTS to version v{this.state.targetKotsVersion}. </p>
              </div>
            }
          </div>
        </div>
      </div>
    );
  }
}

export default withRouter(Footer);
