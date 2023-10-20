import { Component } from "react";
import Modal from "react-modal";
import AceEditor from "react-ace";
import yaml from "js-yaml";
import Loader from "../shared/Loader";
import { Utilities } from "../../utilities/utilities";
import "brace/mode/text";
import "brace/mode/yaml";
import "brace/theme/chrome";
import Icon from "../Icon";

const CUSTOM_SPEC_TEMPLATE = `
apiVersion: troubleshoot.sh/v1beta2
kind: Redactor
metadata:
  name: my-application-name
spec:
  redactors:
  - name: example replacement
    values:
    - abc123
`;

export default class ConfigureRedactorsModal extends Component {
  state = {
    activeRedactorTab: "linkSpec",
    redactorUri: "",
    customRedactorSpec: CUSTOM_SPEC_TEMPLATE,
    errorSavingSpecUri: false,
    specSaved: false,
    errorSavingSpec: false,
    savingSpecUriError: "",
    savingSpecError: "",
    savingRedactor: false,
  };

  toggleRedactorAction = (active) => {
    this.setState({
      activeRedactorTab: active,
      specSaved: false,
      errorSavingSpec: false,
      savingSpecError: "",
      savingSpecUriError: "",
    });
  };

  onRedactorChange = (value) => {
    this.setState({
      customRedactorSpec: value,
    });
  };

  getRedactor = () => {
    this.setState({ loadingRedactor: true });
    fetch(`${process.env.API_ENDPOINT}/redact/get`, {
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
      method: "GET",
    })
      .then(async (res) => {
        const response = await res.json();
        try {
          const r = yaml.safeLoad(response.updatedSpec);
          if (typeof r === "object") {
            this.setState({
              customRedactorSpec: response.updatedSpec,
              showRedactors: response.updatedSpec !== "",
              activeRedactorTab: "writeSpec",
              loadingRedactor: false,
            });
          } else {
            this.setState({
              redactorUri: response.updatedSpec,
              showRedactors: response.updatedSpec !== "",
              activeRedactorTab: "linkSpec",
              loadingRedactor: false,
            });
          }
        } catch (e) {
          console.log(e);
          this.setState({ loadingRedactor: false, errFetchingRedactors: true });
        }
      })
      .catch(() => {
        this.setState({ loadingRedactor: false, errFetchingRedactors: true });
      });
  };

  saveRedactor = () => {
    const { activeRedactorTab, redactorUri, customRedactorSpec } = this.state;
    const isRedactorLink = activeRedactorTab === "linkSpec";
    this.setState({
      errorSavingSpec: false,
      savingSpecError: "",
      savingSpecUriError: "",
    });

    let payload;
    if (isRedactorLink) {
      if (!redactorUri.length || redactorUri === "") {
        return this.setState({
          errorSavingSpecUri: true,
          savingSpecUriError: "No uri was provided",
        });
      }
      payload = {
        redactSpecUrl: redactorUri,
      };
    } else {
      payload = {
        redactSpec: customRedactorSpec,
      };
    }

    this.setState({ savingRedactor: true });
    fetch(`${process.env.API_ENDPOINT}/redact/set`, {
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      credentials: "include",
      method: "PUT",
      body: JSON.stringify(payload),
    })
      .then(async (res) => {
        const body = await res.json();
        if (!res.ok) {
          if (isRedactorLink) {
            this.setState({
              savingRedactor: false,
              errorSavingSpecUri: true,
              savingSpecUriError: body.error,
            });
          } else {
            this.setState({
              savingRedactor: false,
              errorSavingSpec: true,
              savingSpecError: body.error,
            });
          }
        } else {
          if (isRedactorLink) {
            this.setState({ savingRedactor: false, specSaved: true });
          } else {
            this.setState({
              customRedactorSpec: body.updatedSpec,
              savingRedactor: false,
              specSaved: true,
            });
          }
          setTimeout(() => {
            this.setState({ specSaved: false });
          }, 3000);
        }
      })
      .catch((err) => {
        if (isRedactorLink) {
          this.setState({
            savingRedactor: false,
            errorSavingSpecUri: true,
            savingSpecUriError: err,
          });
        } else {
          this.setState({
            savingRedactor: false,
            errorSavingSpec: true,
            savingSpecError: err,
          });
        }
      });
  };

  renderRedactorTab = () => {
    const {
      activeRedactorTab,
      redactorUri,
      customRedactorSpec,
      savingRedactor,
    } = this.state;
    switch (activeRedactorTab) {
      case "linkSpec":
        return (
          <div className="flex1">
            <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--5">
              Where is your spec located
            </p>
            <p className="u-lineHeight--normal u-fontSize--small u-textColor--bodyCopy u-fontWeight--medium u-marginBottom--10">
              Provide the URI where your redactor spec is located.
            </p>
            <input
              type="text"
              className="Input"
              placeholder="github.com/org/myrepo/redactor.yaml"
              value={redactorUri}
              autoComplete=""
              onChange={(e) => {
                this.setState({ redactorUri: e.target.value });
              }}
            />
            <div className="u-marginTop--10 flex alignItems--center">
              <button
                className="btn secondary blue u-marginRight--10"
                onClick={this.props.onClose}
              >
                Close
              </button>
              <button
                className="btn primary"
                onClick={this.saveRedactor}
                disabled={savingRedactor}
              >
                {savingRedactor ? "Saving" : "Save"}
              </button>
              {this.state.specSaved && (
                <span className="u-marginLeft--10 flex alignItems--center">
                  <Icon
                    icon="check-circle-filled"
                    size={16}
                    className="success-color u-marginRight--5"
                  />
                  <span className="u-textColor--success u-fontSize--small u-fontWeight--medium u-lineHeight--normal">
                    Saved
                  </span>
                </span>
              )}
              {this.state.errorSavingSpecUri && (
                <span className="u-marginLeft--10 flex alignItems--center">
                  <span className="u-textColor--error u-fontSize--small u-fontWeight--medium u-lineHeight--normal">
                    {this.state.savingSpecUriError}
                  </span>
                </span>
              )}
            </div>
          </div>
        );
      case "writeSpec":
        return (
          <div>
            <div className="flex1 u-border--gray">
              <AceEditor
                ref={(input) => (this.refAceEditor = input)}
                mode="yaml"
                theme="chrome"
                className="flex1 flex"
                value={customRedactorSpec}
                height="380px"
                width="100%"
                markers={this.state.activeMarkers}
                editorProps={{
                  $blockScrolling: Infinity,
                  useSoftTabs: true,
                  tabSize: 2,
                }}
                onChange={(value) => this.onRedactorChange(value)}
                setOptions={{
                  scrollPastEnd: false,
                  showGutter: true,
                }}
              />
            </div>
            <div className="u-marginTop--10 flex alignItems--center">
              <button
                className="btn secondary blue u-marginRight--10"
                onClick={this.props.onClose}
              >
                Close
              </button>
              <button
                className="btn primary"
                onClick={this.saveRedactor}
                disabled={savingRedactor}
              >
                {savingRedactor ? "Saving spec" : "Save spec"}
              </button>
              {this.state.specSaved && (
                <span className="u-marginLeft--10 flex alignItems--center">
                  <Icon
                    icon="check-circle-filled"
                    size={16}
                    className="success-color u-marginRight--5"
                  />
                  <span className="u-textColor--success u-fontSize--small u-fontWeight--medium u-lineHeight--normal">
                    Spec saved
                  </span>
                </span>
              )}
              {this.state.errorSavingSpec && (
                <span className="u-marginLeft--10 flex alignItems--center">
                  <span className="u-textColor--error u-fontSize--small u-fontWeight--medium u-lineHeight--normal">
                    {this.state.savingSpecError}
                  </span>
                </span>
              )}
            </div>
          </div>
        );
      default:
        return null;
    }
  };

  componentDidMount = () => {
    this.getRedactor();
  };

  render() {
    const { onClose } = this.props;

    return (
      <Modal
        isOpen={true}
        onRequestClose={onClose}
        shouldReturnFocusAfterClose={false}
        contentLabel="Configure redactors modal"
        ariaHideApp={false}
        className={`Modal ${
          this.state.activeRedactorTab === "linkSpec"
            ? "SmallSize"
            : "MediumSize"
        }`}
      >
        <div className="Modal-body">
          <p className="u-fontSize--largest u-fontWeight--bold u-lineHeight--default u-textColor--primary u-marginBottom--small">
            Configure redaction
          </p>
          {this.state.errFetchingRedactors ? (
            <div className="u-marginTop--40 flex justifyContent--center">
              <span className="u-fontSize--large u-fontWeight--medium u-textColor--error u-lineHeight--normal">
                Failed to fetch custom redactors
              </span>
            </div>
          ) : (
            <div className="u-marginTop--40">
              <div className="flex action-tab-bar">
                <span
                  className={`${
                    this.state.activeRedactorTab === "linkSpec"
                      ? "is-active"
                      : ""
                  } tab-item`}
                  onClick={() => this.toggleRedactorAction("linkSpec")}
                >
                  Link to a spec
                </span>
                <span
                  className={`${
                    this.state.activeRedactorTab === "writeSpec"
                      ? "is-active"
                      : ""
                  } tab-item`}
                  onClick={() => this.toggleRedactorAction("writeSpec")}
                >
                  Write your own spec
                </span>
              </div>
              <div className="flex-column flex1 action-content old">
                {this.state.loadingRedactor ? (
                  <div className="flex1 flex-column justifyContent--center alignItems--center">
                    <Loader size="60" />
                  </div>
                ) : (
                  this.renderRedactorTab()
                )}
              </div>
            </div>
          )}
        </div>
      </Modal>
    );
  }
}
