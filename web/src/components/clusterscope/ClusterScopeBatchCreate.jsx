import * as React from "react";
import { Utilities } from "../../utilities/utilities";

import "../../scss/components/image_check/ImageWatchBatch.scss";
import "../../scss/components/Login.scss";
import clusterScopeImageSrc from "../../images/ship-clusterscope@2x.jpg"

class ClusterScopeBatchCreate extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      imageInput: "",
      saving: false,
      serverError: false,
      serverErrorMessage: "",
    }
  }

  onImageInputChanged = (ev) => {
    this.setState({
      imageInput: ev.target.value && ev.target.value.trim(),
    });
  }

  handleSave = () => {
    this.setState({ saving: true });

    this.props.uploadImageWatchBatch(this.state.imageInput)
      .then((result) => {
        this.setState({ saving: false });
        if (!Utilities.isLoggedIn()) {sessionStorage.setItem("showSignUpModal", true);}
        this.props.history.replace(`/clusterscope/${result.data.uploadImageWatchBatch}`)
      })
      .catch((err) => {
        err.graphQLErrors.map(({ message }) => {
          this.setState({
            saving: false,
            serverEror: true,
            serverErrorMessage: message
          });
        });
      });
  }

  render() {
    return (
      <div className="u-overflow--auto ClusterScopePage--wrapper">
        <div className="Login-wrapper flex1 flex-column alignItems--center justifyContent--center container">
          <div className="ClusterScope-create--wrapper justifyContent--spaceBetween alignItems--center u-textAlign--center flex-column flex1 u-paddingTop--30 u-marginTop--10 u-paddingBottom--30">
            <div className="icon kub-logo"></div>
            <p className="u-fontSize--header2 u-marginTop--normal u-fontWeight--bold u-color--tuna u-lineHeight--more">Discover outdated images in your cluster</p>
            <p className="u-fontSize--large u-color--dustyGray u-fontWeight--medium u-lineHeight--more u-marginTop--small">
              Run the standard <code>kubectl</code> command below to extract your image names and SHAs. Submit the output and ClusterScope will analyze each 3rd-party image and provide a shareable report on how current each version is.
            </p>
          </div>
        </div>
        <div className="Output--wrapper container flex-column flex1 container">
          <p className="flex alignItems--center u-fontSize--large u-color--tuna u-fontWeight--bold"><span className="step-count">1</span>Run the command</p>
          <code className="u-lineHeight--normal u-fontSize--small">
            {`kubectl get pods --all-namespaces \\
  -o jsonpath='{range .items[*]}{@.spec.containers[*].image}{","}{@.status.containerStatuses[*].imageID}{"\\n"}{end}' \\
  | sort -u`}
          </code>
          <p className="flex u-marginTop--15 alignItems--center u-fontSize--large u-color--tuna u-fontWeight--bold"><span className="step-count">2</span>Paste your output</p>
          <div className="helm-values flex1 flex u-height--full u-width--full u-marginTop--15">
            <div className="flex1 flex-column u-width--half u-overflow--hidden">
              <textarea
                className="Textarea"
                ref={(editor) => { this.monacoEditor = editor }}
                onChange={this.onImageInputChanged}
                value={this.state.imageInput}
                style={{height: "120px", fontFamily: "monospace"}}
                width="100%"
              />
            </div>
          </div>
          <div className="flex flex1 u-marginBottom--30">
            <p className="flex u-marginTop--15 alignItems--center u-fontSize--large u-color--tuna u-fontWeight--bold"><span className="step-count">3</span>Get a shareable report</p>
            <div className="flex-column flex1 justifyContent--flexEnd">
              <div>
                {this.state.serverEror &&
                <div className="flex-column flex-verticalCenter">
                  <p className="u-color--chestnut u-fontSize--small u-fontWeight--medium u-marginRight--10 u-lineHeight--normal">
                    {this.state.serverErrorMessage}</p>
                </div>
                }
                <div className="flex flex1 justifyContent--flexEnd">
                  <button className="btn primary u-marginRight--normal" onClick={this.handleSave} disabled={this.state.saving}>{this.state.saving ? "Checking" : "Check your images"}</button>
                </div>
              </div>
            </div>
          </div>
        </div>
        <div className="clusterscope-section flex-column flex1 u-position--relative u-overflow--hidden">
          <div className="container flex-column flex1 u-zIndex--2">
            <div className="paddingContainer flex-column flex1">
              <div className="flex-column flex1 device-wrapper alignItems--center">
                <div className="sidedevices">
                  <div className="computerwrapper">
                    <div className="computer">
                      <div className="mask">
                        <img className="mask-img" src={clusterScopeImageSrc} />
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div className="section-gradient"></div>
        </div>
      </div>
    );
  }
}

export default ClusterScopeBatchCreate;

