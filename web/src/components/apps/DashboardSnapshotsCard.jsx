import React from "react";
import { Link, withRouter } from "react-router-dom";
import { Utilities, getSnapshotDestinationLabel } from "@src/utilities/utilities";
import "../../scss/components/watches/DashboardCard.scss";
import InlineDropdown from "../shared/InlineDropdown";
import SnapshotDifferencesModal from "@src/components/modals/SnapshotDifferencesModal";

class DashboardSnapshotsCard extends React.Component {

  state ={
    snapshotSettings: null,
    isLoadingSnapshotSettings: false,
    snapshotSettingsErr: false,
    snapshotSettingsErrMsg: "",
    minimalRBACKotsadmNamespace: "",
  }

  startASnapshot = (option) => {
    const { app } = this.props;
    this.setState({
      startingSnapshot: true,
      startSnapshotErr: false,
      startSnapshotErrorMsg: "",
    });

    let url = option === "full" ?
      `${window.env.API_ENDPOINT}/snapshot/backup`
      : `${window.env.API_ENDPOINT}/app/${app.slug}/snapshot/backup`;

    fetch(url, {
      method: "POST",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      }
    })
      .then(async (result) => {
        if (!result.ok && result.status === 409) {
          const res = await result.json();
          if (res.kotsadmRequiresVeleroAccess) {
            this.setState({
              startingSnapshot: false
            });
            this.props.history.replace("/snapshots/settings");
            return;
          }
        }

        if (result.ok) {
          this.setState({
            startingSnapshot: false
          });
          this.props.ping();
          option === "full" ?
            this.props.history.push("/snapshots")
            : this.props.history.push(`/snapshots/partial/${app.slug}`)
        } else {
          const body = await result.json();
          this.setState({
            startingSnapshot: false,
            startSnapshotErr: true,
            startSnapshotErrorMsg: body.error,
          });
        }
      })
      .catch(err => {
        console.log(err);
        this.setState({
          startSnapshotErrorMsg: err ? err.message : "Something went wrong, please try again."
        });
      })
  }

  fetchSnapshotSettings = async () => {
    this.setState({
      isLoadingSnapshotSettings: true,
      snapshotSettingsErr: false,
      snapshotSettingsErrMsg: "",
      minimalRBACKotsadmNamespace: "",
    });

    fetch(`${window.env.API_ENDPOINT}/snapshots/settings`, {
      method: "GET",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      }
    })
      .then(async res => {
        if (!res.ok && res.status === 409) {
          const result = await res.json();
          if (result.kotsadmRequiresVeleroAccess) {
            this.setState({ isLoadingSnapshotSettings: false });
            // requires velero access so do something here to show that
            // this.openConfigureSnapshotsMinimalRBACModal(result.kotsadmRequiresVeleroAccess, result.kotsadmNamespace);
            return;
          }
        }

        const result = await res.json();
        this.setState({
          snapshotSettings: result,
          kotsadmRequiresVeleroAccess: false,
          isLoadingSnapshotSettings: false,
          snapshotSettingsErr: false,
          snapshotSettingsErrMsg: "",
        });
      })
      .catch(err => {
        this.setState({
          isLoadingSnapshotSettings: false,
          snapshotSettingsErr: true,
          snapshotSettingsErrMsg: err,
        })
      })
  }
  
  toggleSnaphotDifferencesModal = () => {
    this.setState({ snapshotDifferencesModal: !this.state.snapshotDifferencesModal });
  }

  componentDidMount() {
    this.fetchSnapshotSettings();
  }

  render() {
    const { isSnapshotAllowed } = this.props;
    const { snapshotSettings } = this.state;
    const storagePathStr = `${snapshotSettings?.store?.bucket}${snapshotSettings?.store?.path ? `/${snapshotSettings?.store?.path}` : ""}`;
    
    return (
      <div className="flex-column flex1 dashboard-card">
        <div className="flex flex1 justifyContent--spaceBetween alignItems--center">
          <p className="u-fontSize--large u-textColor--primary u-fontWeight--bold">Snapshots</p>
          <div className="u-fontSize--small u-fontWeight--medium flex flex-auto alignItems--center">
            <Link className="replicated-link u-marginRight--20 flex alignItems--center" to="/snapshots/settings">
              <span className="icon clickable dashboard-card-settings-icon u-marginRight--5" />Snapshot settings
            </Link>
            <span className="icon clickable dashboard-card-snapshot-icon u-marginRight--5" />
            <InlineDropdown
              defaultDisplayText="Start snapshot"
              dropdownOptions={[
                { displayText: "Start a Partial snapshot", onClick: () => this.startASnapshot("partial") },
                { displayText: "Start a Full snapshot", onClick: () => this.startASnapshot("full")},
                { displayText: "Learn about the difference", onClick: () => this.toggleSnaphotDifferencesModal() }
              ]}
            />
          </div>
        </div>
        <div className="LicenseCard-content--wrapper u-marginTop--10 flex flex1">
          <div className="flex1">
            <span className={`status-dot ${isSnapshotAllowed ? "u-color--success" : "u-color--warning"}`}/>
            <span className={`u-fontSize--small u-fontWeight--medium ${isSnapshotAllowed ? "u-textColor--success" : "u-textColor--warning"}`}>
              {isSnapshotAllowed ? "Enabled" : "Disabled"}
            </span>
            <div className="flex alignItems--center u-marginTop--10">
              <span className={`icon snapshotDestination--${snapshotSettings?.store?.provider} u-marginRight--5`} />
              <p className="u-fontSize--normal u-fontWeight--medium u-textColor--header">{getSnapshotDestinationLabel(snapshotSettings?.store?.provider)}</p>
            </div>
            <p className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy u-marginTop--10">{storagePathStr}</p>
          </div>
          <div className="flex-auto">
            <div className="u-color--taupe u-padding--10">
              <p></p>
            </div>
          </div>
        </div>
        <div className="u-marginTop--10">
          <Link to={`/snapshots`} className="replicated-link has-arrow u-fontSize--small">See all snapshots</Link>
        </div>
        {this.state.snapshotDifferencesModal &&
          <SnapshotDifferencesModal
            snapshotDifferencesModal={this.state.snapshotDifferencesModal}
            toggleSnapshotDifferencesModal={this.toggleSnaphotDifferencesModal}
          />
        }
      </div>
    );
  }
}

export default withRouter(DashboardSnapshotsCard)