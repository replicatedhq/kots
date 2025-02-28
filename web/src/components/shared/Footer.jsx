import { Component } from "react";
import semverjs from "semver";
import { getBuildVersion } from "@src/utilities/utilities";
import "@src/scss/components/shared/Footer.scss";
import Icon from "../Icon";

export class Footer extends Component {
  state = {
    targetKotsVersion: "",
  };

  componentDidMount() {
    this.setState({ targetKotsVersion: this.getHighestTargetKotsVersion() });
  }

  componentDidUpdate(lastProps) {
    if (this.props.appsList !== lastProps.appsList) {
      this.setState({ targetKotsVersion: this.getHighestTargetKotsVersion() });
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
        targetKotsVersions.push(app.targetKotsVersion);
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
    } catch (err) {
      console.log(err);
    }
  };

  render() {
    return (
      <div
        className={`FooterContent-wrapper flex flex-auto justifyContent--center ${
          this.state.targetKotsVersion && "u-padding--5"
        } ${this.props.className || ""}`}
      >
        <div className="container flex1 flex justifyContent--center alignItems--center">
          <div className="FooterItem-wrapper">
            <span className="FooterItem" data-testid="build-version">{getBuildVersion()}</span>
          </div>
          {this.state.targetKotsVersion && (
            <div className="TargetKotsVersionWrapper flex u-marginLeft--10">
              <Icon
                icon="megaphone-filled"
                size={28}
                className="flex u-marginRight--10 gray-color"
              />
              <p className="u-fontSize--small u-fontWeight--bold">
                {" "}
                v{this.state.targetKotsVersion} available.{" "}
              </p>
            </div>
          )}
        </div>
      </div>
    );
  }
}

export default Footer;
