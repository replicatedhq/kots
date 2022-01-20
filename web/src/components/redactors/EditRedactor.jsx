import React, { Component } from "react";
import { withRouter, Link } from "react-router-dom"
import Helmet from "react-helmet";
import AceEditor from "react-ace";
import "brace/mode/text";
import "brace/mode/yaml";
import "brace/theme/chrome";

import Loader from "../shared/Loader";
import { Utilities } from "../../utilities/utilities";

import "../../scss/components/redactors/EditRedactor.scss"

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

    fetch(`${process.env.API_ENDPOINT}/redact/spec/${slug}`, {
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
      enabled: enabled,
      redactor: yaml
    }

    fetch(`${process.env.API_ENDPOINT}/redact/spec/${slug}`, {
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
          this.props.history.replace(`/app/${this.props.appSlug}/troubleshoot/redactors`)
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
          editingErrMsg: err.message ? err.message : "Something went wrong, please try again."
        });
      });
  }

  getEmptyNameLine = (redactorYaml) => {
    const splittedYaml = redactorYaml.split("\n");
    let metadataFound = false;
    let namePosition;
    for(let i=0; i<splittedYaml.length; ++i) {
      if (splittedYaml[i] === "metadata:") {
        metadataFound = true;
      }
      if (metadataFound && splittedYaml[i].includes("name:")) {
        namePosition = i + 1;
        break;
      }
    }
    return namePosition;
  }

  createRedactor = (enabled, newRedactor, yaml) => {
    this.setState({ creatingRedactor: true, createErrMsg: "" });

    const payload = {
      enabled: enabled,
      new: newRedactor,
      redactor: yaml
    }

    fetch(`${process.env.API_ENDPOINT}/redact/spec/new`, {
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
          });
          const editor = this.aceEditor.editor;
          editor.scrollToLine(this.getEmptyNameLine(this.state.redactorYaml), true, true);
          editor.gotoLine(this.getEmptyNameLine(this.state.redactorYaml), 1, true);
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
          this.props.history.replace(`/app/${this.props.appSlug}/troubleshoot/redactors`)
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
          createErrMsg: err.message ? err.message : "Something went wrong, please try again."
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
      const defaultYaml = `kind: Redactor
apiVersion: troubleshoot.sh/v1beta2
metadata:
  name: 
spec:
  redactors:
  - name: myredactor
    fileSelector:
      files:
      - "abc"
    removals:
      values:
      - "removethis"`
      this.setState({ redactorEnabled: true, redactorYaml: defaultYaml, redactorName: "New redactor" });
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
    const { isLoadingRedactor, createConfirm, editConfirm, creatingRedactor, editingRedactor, createErrMsg, editingErrMsg } = this.state;

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
          {(createErrMsg || editingErrMsg) && <p className="ErrorToast flex justifyContent--center alignItems--center">{createErrMsg ? createErrMsg : editingErrMsg}</p>}
          <div className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy u-marginBottom--20">
            <Link to={`/app/${this.props.appSlug}/troubleshoot/redactors`} className="replicated-link u-marginRight--5">Redactors</Link> &gt; <span className="u-marginLeft--5">{this.state.redactorName}</span>
          </div>
          <div className="flex flex-auto alignItems--flexStart justifyContent--spaceBetween">
            <div className="flex flex1 alignItems--center">
              <p className="u-fontWeight--bold u-textColor--primary u-fontSize--jumbo u-lineHeight--normal u-marginRight--10">{this.state.redactorName}</p>
            </div>
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
                  <p className="u-fontWeight--medium u-textColor--secondary u-fontSize--large alignSelf--center">{this.state.redactorEnabled ? "Enabled" : "Disabled"}</p>
                </div>
              </div>
            </div>
          </div>
          <p className="u-fontSize--normal u-textColor--bodyCopy u-fontWeight--medium u-lineHeight--normal u-marginTop--10">For more information about creating redactors,
          <a href="https://troubleshoot.sh/reference/redactors/overview/" target="_blank" rel="noopener noreferrer" className="replicated-link"> check out our docs</a>.</p>
          <div className="flex1 u-marginTop--30 u-border--gray">
            <AceEditor
              ref={el => (this.aceEditor = el)}
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
              <Link to={`/app/${this.props.appSlug}/troubleshoot/redactors`} className="btn secondary"> Cancel </Link>
            </div>
            <div className="flex alignItems--center">
              {createConfirm || editConfirm &&
                <div className="u-marginRight--10 flex alignItems--center">
                  <span className="icon checkmark-icon" />
                  <span className="u-marginLeft--5 u-fontSize--small u-fontWeight--medium u-textColor--success">{createConfirm ? "Redactor created" : "Redactor updated"}</span>
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

export default withRouter(EditRedactor);
