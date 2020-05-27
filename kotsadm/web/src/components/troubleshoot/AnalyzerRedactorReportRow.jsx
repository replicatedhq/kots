
import React from "react";
import { Link } from "react-router-dom"
import groupBy from "lodash/groupBy";

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

  goToFile = (filePath) => {
    const { match, history, redactorFiles } = this.props;
    const filteredFiles = redactorFiles.filter(f => f.file === filePath);
    let rowString = "#";
    filteredFiles.forEach((file, i) => {
      if (i === 0) {
        rowString = `${rowString}${file.line}`;
      } else {
        rowString = `${rowString},${file.line}`;
      }
    });
    history.push(`/app/${match.params.slug}/troubleshoot/analyze/${match.params.bundleSlug}/contents/${filePath}${rowString}`);
  }

  renderRedactorFiles = (file, totalFileRedactions, i) => {
    return (
      <div className="flex flex1 alignItems--center section u-marginTop--10" key={`${file.file}-${i}`}>
        <div className="flex u-marginRight--10">
          <span className={`icon redactor-${this.getRedactorExtension(file?.file)}-icon`} />
        </div>
        <div className="flex flex-column">
          <p className="u-fontSize--large u-lineHeight--normal u-fontWeight--bold u-color--tuna">{this.calculateRedactorFileName(file?.file)} <span className="u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-color--chateauGreen"> {totalFileRedactions} redaction{totalFileRedactions === 1 ? "" : "s"} </span> </p>
          <div className="flex flex1 alignItems--center u-cursor--pointer" onClick={() => this.goToFile(file?.file)}>
            <p className="u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-color--dustyGray"> {file?.file} </p>
            <div className="icon u-iconFullArrowGray" />
          </div>
        </div>
      </div>
    )
  }

  renderRedactionDetails = (files, totalLength) => {
    if (totalLength > 0) {
      return (
        <div className="flex flex1 alignItems--center">
          <p className="u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-color--dustyGray"> <span className="u-color--chateauGreen"> {totalLength} redaction{totalLength === 1 ? "" : "s"} </span> across <span className="u-color--nevada">{files?.length} file{files?.length === 1 ? "" : "s"}</span></p>
          <span className="replicated-link u-fontSize--small u-marginLeft--10" onClick={this.toggleDetails}> {this.state.toggleDetails ? "Hide details" : "Show details"} </span>
        </div>
      )
    } else {
      return <p className="u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-color--dustyGray"> This redactor doesn't have any additional files</p>
    }
  }


  render() {
    const { redactor, redactorFiles } = this.props;
    const groupedFiles = groupBy(redactorFiles, "file")
    const groupedFilesArray = Object.keys(groupedFiles).map(i => groupedFiles[i]);

    return (
      <div className="flex flex-auto ActiveDownstreamVersionRow--wrapper" key={redactor}>
        <div className="flex flex1 alignItems--center">
          <div className="flex flex-column">
            <div className="flex flex1 alignItems--center">
              <span className="icon redactor-yaml-icon u-marginRight--10" />
              <div className="flex flex-column">
                <p className="u-fontSize--large u-lineHeight--normal u-fontWeight--bold u-color--tuna">{redactor}</p>
                {this.renderRedactionDetails(groupedFilesArray, redactorFiles.length)}
              </div>
            </div>
            {this.state.toggleDetails &&
              <div className="Timeline--wrapper" style={{ marginLeft: "43px" }}>
                {groupedFilesArray.length > 0 && groupedFilesArray?.map((file, i) => {
                  return this.renderRedactorFiles(file[0], file.length, i)
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
