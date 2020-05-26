
import React from "react";

class AnalyzerRedactorReportRow extends React.Component {
  state = {
    redactorEnabled: true,
    toggleDetails: false
  };

  handleEnableRedactor = () => {
    this.setState({
      redactorEnabled: !this.state.redactorEnabled,
    });
  }

  toggleDetails = () => {
    this.setState({ toggleDetails: !this.state.toggleDetails });
  }

  calculateRedactorFileName = (file) => {
    const splitFile = file.split("/");
    return splitFile.pop();
  }

  getRedactorExtension = (file) => {
    const extensionFile = this.calculateRedactorFileName(file);

    if (extensionFile.split(".").pop() !== "yaml" && extensionFile.split(".").pop() !== "json") {
      return "text";
    } else {
      return extensionFile.split(".").pop();
    }
  }

  renderRedactorFiles = (file, i) => {
    return (
      <div className="flex flex1 alignItems--center section u-marginTop--10" key={`${file.file}-${i}`}>
        <div className="flex u-marginRight--10">
          <span className={`icon redactor-${this.getRedactorExtension(file?.file)}-icon`} />
        </div>
        <div className="flex flex-column">
          <p className="u-fontSize--large u-lineHeight--normal u-fontWeight--bold u-color--tuna">{this.calculateRedactorFileName(file?.file)} <span className="u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-color--chateauGreen"> 1 redaction </span> </p>
            <div className="flex flex1 alignItems--center">
              <p className="u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-color--dustyGray"> {file?.file} </p>
            </div>
        </div>
      </div>
    )
  }

  renderRedactionDetails = (files) => {
    if (files.length > 0) {
      return (
        <div className="flex flex1 alignItems--center">
          <p className="u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-color--dustyGray"> <span className="u-color--chateauGreen"> {files?.length} redactions </span> across <span className="u-color--nevada">{files?.length} files</span></p>
          <span className="replicated-link u-fontSize--small u-marginLeft--10" onClick={this.toggleDetails}> {this.state.toggleDetails ? "Hide details" : "Show details"} </span>
        </div>
      )
    } else {
      return <p className="u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-color--dustyGray"> This redactor doesn't have any additional files</p>
    }
  }


  render() {
    const { redactor, redactorFiles } = this.props;

    return (
      <div className="flex flex-auto ActiveDownstreamVersionRow--wrapper" key={redactor}>
        <div className="flex flex1 alignItems--center">
          <div className="flex flex-column">
            <div className="flex flex1 alignItems--center">
              <span className="icon redactor-yaml-icon u-marginRight--10" />
              <div className="flex flex-column">
                <p className="u-fontSize--large u-lineHeight--normal u-fontWeight--bold u-color--tuna">{redactor}</p>
                {this.renderRedactionDetails(redactorFiles)}
              </div>
            </div>
            {this.state.toggleDetails &&
              <div className="Timeline--wrapper" style={{ marginLeft: "43px" }}>
                {redactorFiles.length > 0 && redactorFiles?.map((file, i) => {
                  return this.renderRedactorFiles(file, i)
                })}
              </div>
            }
          </div>
        </div>
        <div className="flex">
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
      </div>
    )
  }
}

export default AnalyzerRedactorReportRow;
