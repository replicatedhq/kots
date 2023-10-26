import { Component } from "react";
import { Repeater } from "@src/utilities/repeater";
import "@src/scss/components/AirgapUploadProgress.scss";

class LicenseUploadProgress extends Component {
  constructor(props) {
    super(props);

    this.state = {
      installStatus: "",
      currentMessage: "",
      getOnlineInstallStatusJob: new Repeater(),
    };
  }

  componentDidMount() {
    this.state.getOnlineInstallStatusJob.start(
      this.getOnlineInstallStatus,
      2000
    );
  }

  componentWillUnmount() {
    this.state.getOnlineInstallStatusJob.stop();
  }

  componentDidUpdate(lastProps, lastState) {
    const { installStatus } = this.state;
    const { onError } = this.props;
    if (
      installStatus !== lastState.installStatus &&
      installStatus === "upload_error"
    ) {
      if (onError && typeof onError === "function") {
        onError(this.state.currentMessage);
      }
    }
  }

  getOnlineInstallStatus = async () => {
    try {
      const res = await fetch(`${process.env.API_ENDPOINT}/app/online/status`, {
        headers: {
          "Content-Type": "application/json",
        },
        method: "GET",
        credentials: "include",
      });

      if (!res.ok) {
        this.setState({
          installStatus: "upload_error",
          currentMessage: `Encountered an error while uploading license: Status ${res.status}`,
        });
      } else {
        const response = await res.json();
        this.setState({
          installStatus: response.installStatus,
          currentMessage: response.currentMessage,
        });
      }
    } catch (err) {
      console.log(err);
      this.setState({
        installStatus: "upload_error",
        currentMessage: err
          ? `Encountered an error while uploading license: ${err.message}`
          : "Something went wrong, please try again.",
      });
    }
  };

  render() {
    let statusDiv = (
      <div
        className={`u-marginTop--10 u-lineHeight--medium u-textAlign--center`}
      >
        <p className="u-textColor--secondary u-fontSize--normal u-fontWeight--bold u-marginBottom--10 u-paddingBottom--5">
          {this.state.currentMessage}
        </p>
        <p className="u-fontSize--small u-textColor--bodyCopy u-fontWeight--medium">
          This may take a while depending on your network connection.
        </p>
      </div>
    );

    return (
      <div className="AirgapUploadProgress--wrapper flex1 flex-column alignItems--center justifyContent--center">
        <div className="flex1 flex-column alignItems--center justifyContent--center u-textColor--primary">
          <p className="u-marginTop--10 u-paddingTop--5 u-marginBottom--5 u-fontSize--header u-textColor--primary u-fontWeight--bold">
            Installing your license
          </p>
          <div className="u-marginTop--20">
            <div className="progressbar medium">
              <div
                id="myBar"
                className="progressbar-meter"
                style={{ width: "0%" }}
              ></div>
            </div>
          </div>
          {statusDiv}
        </div>
      </div>
    );
  }
}

export default LicenseUploadProgress;
