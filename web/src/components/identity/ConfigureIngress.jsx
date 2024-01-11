import { Component } from "react";
import { KotsPageTitle } from "@components/Head";
import Dropzone from "react-dropzone";
import isEmpty from "lodash/isEmpty";

import AnnotationRow from "./AnnotationRow";
import Icon from "../Icon";

class ConfigureIngress extends Component {
  state = {
    hostname: "",
    secretName: "",
    certFile: {},
    ingressEnabled: true,
    showAdvancedOptions: true,
    annotationRowsNum: 1,
  };

  handleFormChange = (field, e) => {
    let nextState = {};
    if (field === "ingressEnabled") {
      nextState[field] = e.target.checked;
    } else {
      nextState[field] = e.target.value;
    }
    this.setState(nextState);
  };

  onDrop = () => {
    // TODO
    this.setState({ certFile: {} });
  };

  toggleAdvancedOptions = () => {
    this.setState({ showAdvancedOptions: !this.state.showAdvancedOptions });
  };

  addAnnotation = () => {
    this.setState({ annotationRowsNum: this.state.annotationRowsNum + 1 });
  };

  removeAnnotation = () => {
    this.setState({ annotationRowsNum: this.state.annotationRowsNum - 1 });
  };

  render() {
    const { certFile } = this.state;
    const hasFile = !isEmpty(certFile);

    const annotationRows = [];

    for (let i = 0; i < this.state.annotationRowsNum; ++i) {
      annotationRows.push(
        <AnnotationRow
          key={i}
          number={i}
          removeAnnotation={this.removeAnnotation}
        />
      );
    }

    return (
      <div className="flex-column flex1 u-position--relative u-overflow--auto u-padding--20 alignItems--center">
        <KotsPageTitle pageName="Configure Ingress" showAppSlug />
        <form className="flex flex-column Identity--wrapper u-marginTop--30">
          <p className="u-fontSize--largest u-lineHeight--default u-fontWeight--bold u-textColor--primary">
            {" "}
            Configure Ingress for Admin Console{" "}
          </p>

          <div className="BoxedCheckbox-wrapper flex1 u-textAlign--left u-marginTop--20">
            <div
              className={`flex-auto flex ${
                this.state.ingressEnabled ? "is-active" : ""
              }`}
            >
              <input
                type="checkbox"
                className="u-cursor--pointer"
                id="ingressEnabled"
                checked={this.state.ingressEnabled}
                onChange={(e) => {
                  this.handleFormChange("ingressEnabled", e);
                }}
              />
              <label
                htmlFor="ingressEnabled"
                className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none"
                style={{ marginTop: "2px" }}
              >
                <div className="flex flex-column u-marginLeft--5 justifyContent--center">
                  <p className="u-textColor--primary u-fontSize--normal u-fontWeight--medium">
                    Enable Ingress for Admin Console
                  </p>
                  <p className="u-fontSize--normal u-lineHeight--normal u-fontWeight--normal u-marginTop--5">
                    {" "}
                    You can configure your own Ingress by unchecking this box.{" "}
                  </p>
                </div>
              </label>
            </div>
          </div>

          {this.state.ingressEnabled && (
            <div className="flex flex-column">
              <div className="u-marginTop--30">
                <div className="flex flex1 alignItems--center">
                  <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-textColor--primary">
                    {" "}
                    Hostname{" "}
                  </p>
                  <span className="required-label"> Required </span>
                </div>
                <p className="u-fontSize--normal u-lineHeight--normal u-fontWeight--medium u-marginTop--5">
                  {" "}
                  This is the host at which you can reach the admin console.{" "}
                </p>
                <input
                  type="text"
                  className="Input u-marginTop--12 u-marginBottom--5"
                  placeholder="kots.somebigbankadmin.com"
                  value={this.state.hostname}
                  onChange={(e) => {
                    this.handleFormChange("hostname", e);
                  }}
                />
                <span className="u-fontSize--small u-fontWeight--medium u-textColor--info">
                  {" "}
                  This hostname must be resolvable to your cluster.{" "}
                </span>
              </div>

              <div className="u-marginTop--30">
                <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-textColor--primary">
                  {" "}
                  TLS{" "}
                </p>
                <p className="u-fontSize--normal u-lineHeight--normal u-fontWeight--medium u-marginTop--5">
                  {" "}
                  Upload or reference your TLS secret file{" "}
                </p>
                <div
                  className={`FileUpload-wrapper flex justifyContent--center u-marginTop--12 ${
                    hasFile ? "has-file" : ""
                  }`}
                >
                  <Dropzone
                    className="Dropzone-wrapper"
                    accept={["application/xml", ".pem", ".cer", ".crt", ".key"]}
                    onDropAccepted={this.onDrop}
                    multiple={false}
                  >
                    <div className="flex flex1">
                      <span className="icon drag-file" />
                      {hasFile ? (
                        <div className="has-file-wrapper">
                          <p className="u-fontSize--normal u-fontWeight--medium">
                            {certFile?.name}
                          </p>
                        </div>
                      ) : (
                        <div className="flex-column">
                          <p className="u-fontSize--normal u-textColor--secondary u-fontWeight--medium u-lineHeight--normal">
                            Drag your cert here or{" "}
                            <span className="link u-textDecoration--underlineOnHover">
                              choose a file
                            </span>
                          </p>
                          <p className="u-fontSize--small u-textColor--info u-fontWeight--normal u-lineHeight--normal">
                            Supported file types are .pem .cer .crt and .key{" "}
                          </p>
                        </div>
                      )}
                    </div>
                  </Dropzone>
                </div>
                <p className="u-fontSize--normal u-lineHeight--normal u-fontWeight--medium u-marginTop--5">
                  {" "}
                  Optionally you can reference the TLS secret deployed to your
                  Cluster.{" "}
                </p>
                <input
                  type="text"
                  className="Input u-marginTop--12"
                  placeholder="secretName"
                  value={this.state.secretName}
                  onChange={(e) => {
                    this.handleFormChange("secretName", e);
                  }}
                />
              </div>

              <div className="u-marginTop--20">
                <p
                  className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal link"
                  onClick={this.toggleAdvancedOptions}
                >
                  {" "}
                  Advanced options <span />{" "}
                </p>
                {this.state.showAdvancedOptions && (
                  <div className="u-marginTop--12">
                    <div className="flex flex-column u-borderBottom--gray darker">
                      <p className="u-fontSize--large u-lineHeight--default u-fontWeight--bold u-textColor--primary">
                        {" "}
                        Anotations{" "}
                      </p>
                      <p className="u-fontSize--normal u-lineHeight--normal u-fontWeight--medium u-marginTop--5 u-marginBottom--10">
                        {" "}
                        Add any required annotations for your configuration.{" "}
                      </p>
                    </div>
                    {this.state.showAdvancedOptions && annotationRows}
                    <p
                      className="u-fontSize--small u-lineHeight--normal u-marginTop--15 link"
                      onClick={this.addAnnotation}
                    >
                      {" "}
                      + Add annotation{" "}
                    </p>
                  </div>
                )}
              </div>

              <div className="flex flex-column u-marginTop--40 flex">
                {this.state.savingIngressErrMsg && (
                  <div className="u-marginBottom--10 flex alignItems--center">
                    <span className="u-fontSize--small u-fontWeight--medium u-textColor--error">
                      {this.state.savingIngressErrMsg}
                    </span>
                  </div>
                )}
                <div className="flex flex1">
                  <button
                    className="btn primary blue"
                    disabled={this.state.savingIngress}
                    onClick={this.onSubmit}
                  >
                    {this.state.savingIngress
                      ? "Saving"
                      : "Save ingress settings"}
                  </button>
                  {this.state.saveConfirm && (
                    <div className="u-marginLeft--10 flex alignItems--center">
                      <Icon
                        icon="check-circle-filled"
                        size={16}
                        className="success-color"
                      />
                      <span className="u-marginLeft--5 u-fontSize--small u-fontWeight--medium u-textColor--success">
                        Settings saved
                      </span>
                    </div>
                  )}
                </div>
              </div>
            </div>
          )}
        </form>
      </div>
    );
  }
}

export default ConfigureIngress;
