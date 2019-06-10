import * as React from "react";
import PropTypes from "prop-types";
import WatchContributors from "../WatchContributors";
import Select from "react-select";

export default class CardRightSideBar extends React.Component {
  static propTypes = {
    watch: PropTypes.object.isRequired,
    submitCallback: PropTypes.func
  }

  state = {
    downloadCluster: {
      value: "",
      label: "Select a cluster",
      watchId: ""
    }
  };

  onDownloadClusterChange = (selectedOption) => {
    this.setState({ downloadCluster: selectedOption });
    this.props.setWatchIdToDownload(selectedOption.watchId);
  }

  downloadAssetsForCluster = () => {
    if (this.props.handleDownload && typeof this.props.handleDownload === "function") {
      this.props.handleDownload();
    }
  }

  render() {
    const { watch, submitCallback, childWatches } = this.props;
    let options = childWatches && childWatches.map((childWatch) => {
      const childCluster = childWatch.cluster;
      if (childCluster) {
        return ({
          value: childCluster.id,
          label: childCluster.title,
          watchId: childWatch.id
        });
      } else {
        return {}
      }
    });
    options && options.push({
      value: watch.id,
      label: `${watch.watchName} (mid-stream)`,
      watchId: watch.id
    })

    return (
      <div className="installed-watch-sidebar flex-column u-width--full">
        <div className="contributors flex-column u-marginBottom--20">
          <WatchContributors
            title="contributors"
            contributors={watch.contributors || []}
            watchName={watch.watchName}
            watchId={watch.id}
            slug={watch.slug}
            watchCallback={submitCallback}
          />
        </div>
        <div className="assets flex flex-column">
          <div>
            <p className="uppercase-title">DOWNLOAD ASSETS</p>
            <div className="flex-column">
              <p className="u-fontWeight--bold u-fontSize--normal u-color--tundora">Select a cluster</p>
              <p className="u-fontWeight--medium u-fontSize--small u-color--dustyGray u-marginTop--5 u-lineHeight--medium">Select the cluster you would like to download your Ship YAML assets from.</p>
              <div className="u-marginTop--10">
                <Select
                  className="replicated-select-container"
                  classNamePrefix="replicated-select"
                  options={options}
                  getOptionLabel={(downloadCluster) => downloadCluster.label}
                  value={this.state.downloadCluster}
                  onChange={this.onDownloadClusterChange}
                  isOptionSelected={(option) => {option.value === this.state.downloadCluster.value}}
                />
              </div>
              <div className="u-marginTop--10">
                <button disabled={this.state.downloadCluster.value === ""} onClick={() => this.downloadAssetsForCluster()} className="btn secondary">Download assets</button>
              </div>
            </div>
          </div>

        </div>
      </div>
    );
  }
}
