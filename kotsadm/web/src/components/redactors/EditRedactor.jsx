import React, { Component } from "react";
import { compose, withApollo } from "react-apollo";
import { withRouter, Link } from "react-router-dom"
import Helmet from "react-helmet";
import AceEditor from "react-ace";
import "brace/mode/text";
import "brace/mode/yaml";
import "brace/theme/chrome";

import Loader from "../shared/Loader";
import { Utilities } from "../../utilities/utilities";

class EditRedactor extends Component {
  state = {
    redactorEnabled: false,
    redactorYaml: "",
    redactorName: "",
    creatingRedactor: false,
    createErrMsg: "",
    createConfirm: false,
    editingRedactor: false,
    editingErrMsg: "",
    editConfirm: false,
    isLoadingRedactor: false,
    redactorErrMsg: ""
  };

  getRedactor = (slug) => {
    this.setState({
      isLoadingRedactor: true,
      redactorErrMsg: ""
    });

    fetch(`${window.env.API_ENDPOINT}/redact/spec/${slug}`, {
      method: "GET",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      }
    })
      .then(res => res.json())
      .then(result => {
        if (result.success) {
          this.setState({
            redactorYaml: result.redactor,
            redactorName: result.redactorMetadata.name,
            redactorEnabled: result.redactorMetadata.enabled,
            isLoadingRedactor: false,
            redactorErrMsg: "",
          }, () => {
            if (this.state.selectedOption) {
              this.sortRedactors(this.state.selectedOption.value);
            }
          })
        } else {
          this.setState({
            isLoadingRedactor: false,
            redactorErrMsg: result.error,
          })
        }
      })
      .catch(err => {
        this.setState({
          isLoadingRedactor: false,
          redactorErrMsg: err,
        })
      })
  }

  editRedactor = (slug, enabled, yaml) => {
    this.setState({ editingRedactor: true, editingErrMsg: "" });

    const payload = {
      slug: slug,
      enabled: enabled,
      redactor: yaml
    }

    fetch(`${window.env.API_ENDPOINT}/redact/spec/${slug}`, {
      method: "POST",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload)
    })
      .then(async (res) => {
        const editResponse = await res.json();
        if (!res.ok) {
          this.setState({
            editingRedactor: false,
            editingErrMsg: editResponse.error
          })
          return;
        }

        if (editResponse.success) {
          this.setState({
            redactorYaml: editResponse.redactor,
            redactorName: editResponse.redactorMetadata.name,
            redactorEnabled: editResponse.redactorMetadata.enabled,
            editingRedactor: false,
            editConfirm: true,
            createErrMsg: ""
          });
          setTimeout(() => {
            this.setState({ editConfirm: false })
          }, 3000);
        } else {
          this.setState({
            editingRedactor: false,
            editingErrMsg: editResponse.error
          })
        }
      })
      .catch((err) => {
        this.setState({
          editingRedactor: false,
          editingErrMsg: err.message ? err.message : "Something went wrong, please try again!"
        });
      });
  }

  createRedactor = (enabled, newRedactor, yaml) => {
    this.setState({ creatingRedactor: true, createErrMsg: "" });

    const payload = {
      enabled: enabled,
      new: newRedactor,
      redactor: yaml
    }

    fetch(`${window.env.API_ENDPOINT}/redact/spec/new`, {
      method: "POST",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload)
    })
      .then(async (res) => {
        const createResponse = await res.json();
        if (!res.ok) {
          this.setState({
            creatingRedactor: false,
            createErrMsg: createResponse.error
          })
          return;
        }

        if (createResponse.success) {
          this.setState({
            redactorYaml: createResponse.redactor,
            redactorName: createResponse.redactorMetadata.name,
            redactorEnabled: createResponse.redactorMetadata.enabled,
            creatingRedactor: false,
            createConfirm: true,
            createErrMsg: ""
          });
          setTimeout(() => {
            this.setState({ createConfirm: false })
          }, 3000);
        } else {
          this.setState({
            creatingRedactor: false,
            createErrMsg: createResponse.error
          })
        }
      })
      .catch((err) => {
        this.setState({
          creatingRedactor: false,
          createErrMsg: err.message ? err.message : "Something went wrong, please try again!"
        });
      });
  }

  handleEnableRedactor = () => {
    this.setState({
      redactorEnabled: !this.state.redactorEnabled,
    });
  }

  componentDidMount() {
    if (this.props.match.params.redactorSlug) {
      this.getRedactor(this.props.match.params.redactorSlug);
    } else {
      const defaultYaml = `name: ""
files: []
values: []
regex: []
multiLine: []
yaml: []`
      this.setState({ redactorEnabled: false, redactorYaml: defaultYaml, redactorName: "New redactor" });
    }
  }

  onYamlChange = (value) => {
    this.setState({ redactorYaml: value });
  }

  onSaveRedactor = () => {
    if (this.props.match.params.redactorSlug) {
      this.editRedactor(this.props.match.params.redactorSlug, this.state.redactorEnabled, this.state.redactorYaml);
    } else {
      this.createRedactor(this.state.redactorEnabled, true, this.state.redactorYaml);
    }
  }


  render() {
    const { isLoadingRedactor, createConfirm, editConfirm, creatingRedactor, editingRedactor } = this.state;

    if (isLoadingRedactor) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      )
    }

    return (
      <div className="container flex-column flex1 u-overflow--auto u-paddingTop--30 u-paddingBottom--20 justifyContent--center alignItems--center">
        <Helmet>
          <title>Redactors</title>
        </Helmet>
        <div className="Redactors--wrapper flex1 flex-column u-width--full">
          <div className="u-fontSize--small u-fontWeight--medium u-color--dustyGray u-marginBottom--20">
            <Link to={`/app/${this.props.appSlug}/troubleshoot/redactors`} className="replicated-link u-marginRight--5">Redactors</Link> > <span className="u-marginLeft--5">{this.state.redactorName}</span>
          </div>
          <div className="flex flex-auto alignItems--flexStart justifyContent--spaceBetween">
            <div className="flex flex1 alignItems--center">
              <p className="u-fontWeight--bold u-color--tuna u-fontSize--jumbo u-lineHeight--normal u-marginRight--10">{this.state.redactorName}</p>
            </div>
            {!this.props.isNew &&
              <div className="flex justifyContent--flexEnd">
                <div className="toggle flex flex1">
                  <div className="flex flex1">
                    <div className={`Checkbox--switch ${this.state.redactorEnabled ? "is-checked" : "is-notChecked"}`}>
                      <input
                        type="checkbox"
                        className="Checkbox-toggle"
                        name="isRedactorEnabled"
                        checked={this.state.redactorEnabled}
                        onChange={(e) => { this.handleEnableRedactor(e) }}
                      />
                    </div>
                  </div>
                  <div className="flex flex1 u-marginLeft--5">
                    <p className="u-fontWeight--medium u-color--tundora u-fontSize--large alignSelf--center">{this.state.redactorEnabled ? "Enabled" : "Disabled"}</p>
                  </div>
                </div>
              </div>
            }
          </div>
          <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-lineHeight--normal u-marginTop--small">For more information about creating redactors,
          <a href="https://troubleshoot.sh/reference/redactors/overview/" target="_blank" rel="noopener noreferrer" className="replicated-link"> check out our docs</a>.</p>
          <div className="flex1 u-marginTop--30 u-border--gray">
            <AceEditor
              ref={(input) => this.refAceEditor = input}
              mode="yaml"
              theme="chrome"
              className="flex1 flex"
              value={this.state.redactorYaml}
              height="100%"
              width="100%"
              markers={this.state.activeMarkers}
              editorProps={{
                $blockScrolling: Infinity,
                useSoftTabs: true,
                tabSize: 2,
              }}
              onChange={(value) => this.onYamlChange(value)}
              setOptions={{
                scrollPastEnd: false,
                showGutter: true,
              }}
            />
          </div>
          <div className="flex u-marginTop--20 justifyContent--spaceBetween">
            <div className="flex">
              <Link to="/redactors" className="btn secondary"> Cancel </Link>
            </div>
            <div className="flex">
              {createConfirm || editConfirm &&
                <div className="u-marginRight--10 flex alignItems--center">
                  <span className="icon checkmark-icon" />
                  <span className="u-marginLeft--5 u-fontSize--small u-fontWeight--medium u-color--chateauGreen">{createConfirm ? "Redactor created" : "Redactor updated"}</span>
                </div>
              }
              <button type="button" className="btn primary blue" onClick={this.onSaveRedactor} disabled={creatingRedactor || editingRedactor}>{(creatingRedactor || editingRedactor) ? "Saving" : "Save redactor"} </button>
            </div>
          </div>
        </div>
      </div>
    );
  }
}

export default compose(
  withApollo,
  withRouter,
)(EditRedactor);
