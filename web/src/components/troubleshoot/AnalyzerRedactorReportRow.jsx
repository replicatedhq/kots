import React from "react";
import groupBy from "lodash/groupBy";
import Icon from "../Icon";
import { withRouter } from "@src/utilities/react-router-utilities";

class AnalyzerRedactorReportRow extends React.Component {
  state = {
    toggleDetails: false,
  };

  toggleDetails = () => {
    this.setState({ toggleDetails: !this.state.toggleDetails });
  };

  calculateRedactorFileName = (file) => {
    const splitFile = file.split("/");
    return splitFile.pop();
  };

  getRedactorExtension = (file) => {
    const extensionFile = this.calculateRedactorFileName(file);
    if (
      extensionFile.split(".").pop() !== "yaml" &&
      extensionFile.split(".").pop() !== "json"
    ) {
      return "text";
    } else {
      return extensionFile.split(".").pop();
    }
  };

  goToFile = (filePath) => {
    const { params, navigate, redactorFiles, redactor } = this.props;
    const filteredFiles = redactorFiles.filter((f) => f.file === filePath);
    let rowString = "#";
    filteredFiles.forEach((file, i) => {
      if (i === 0) {
        rowString = `${rowString}${file.line}`;
      } else {
        rowString = `${rowString},${file.line}`;
      }
    });
    navigate(
      `/app/${params.slug}/troubleshoot/analyze/${params.bundleSlug}/contents/${filePath}?file=${redactor}${rowString}`
    );
  };

  renderRedactorFiles = (file, totalFileRedactions, i) => {
    return (
      <div
        className="flex flex1 alignItems--center section u-marginTop--10"
        key={`${file.file}-${i}`}
      >
        <div className="flex u-marginRight--10">
          <span
            className={`icon redactor-${this.getRedactorExtension(
              file?.file
            )}-icon`}
          />
        </div>
        <div className="flex flex-column">
          <p className="u-fontSize--large u-lineHeight--normal u-fontWeight--bold u-textColor--primary">
            {this.calculateRedactorFileName(file?.file)}{" "}
            <span className="u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-textColor--success">
              {" "}
              {totalFileRedactions} redaction
              {totalFileRedactions === 1 ? "" : "s"}{" "}
            </span>{" "}
          </p>
          <div
            className="flex flex1 alignItems--center u-cursor--pointer"
            onClick={() => this.goToFile(file?.file)}
          >
            <p className="u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-textColor--bodyCopy">
              {" "}
              {file?.file}{" "}
            </p>
            <Icon
              icon="right-arrow-pointer"
              size={13}
              className="gray-color u-marginLeft--5"
            />
          </div>
        </div>
      </div>
    );
  };

  renderRedactionDetails = (files, totalLength) => {
    if (totalLength > 0) {
      return (
        <div className="flex flex1 alignItems--center">
          <p className="u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-textColor--bodyCopy">
            {" "}
            <span className="u-textColor--success">
              {" "}
              {totalLength} redaction{totalLength === 1 ? "" : "s"}{" "}
            </span>{" "}
            across{" "}
            <span className="u-textColor--accent">
              {files?.length} file{files?.length === 1 ? "" : "s"}
            </span>
          </p>
          <span
            className="link u-fontSize--small u-marginLeft--10"
            onClick={this.toggleDetails}
          >
            {" "}
            {this.state.toggleDetails ? "Hide details" : "Show details"}{" "}
          </span>
        </div>
      );
    } else {
      return (
        <p className="u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-textColor--bodyCopy">
          {" "}
          This redactor doesn't have any additional files
        </p>
      );
    }
  };

  render() {
    const { redactor, redactorFiles } = this.props;
    const groupedFiles = groupBy(redactorFiles, "file");
    const groupedFilesArray = Object.keys(groupedFiles).map(
      (i) => groupedFiles[i]
    );

    return (
      <div
        className="flex flex-auto RedactorReportRow--wrapper"
        key={redactor}
      >
        <div className="flex flex1 alignItems--center">
          <div className="flex flex-column">
            <div className="flex flex1 alignItems--center">
              <span className="icon redactor-yaml-icon u-marginRight--10" />
              <div className="flex flex-column">
                <p className="u-fontSize--large u-lineHeight--normal u-fontWeight--bold u-textColor--primary">
                  {redactor}
                </p>
                {this.renderRedactionDetails(
                  groupedFilesArray,
                  redactorFiles.length
                )}
              </div>
            </div>
            {this.state.toggleDetails && (
              <div className="Timeline--wrapper" style={{ marginLeft: "43px" }}>
                {groupedFilesArray.length > 0 &&
                  groupedFilesArray?.map((file, i) => {
                    return this.renderRedactorFiles(file[0], file.length, i);
                  })}
              </div>
            )}
          </div>
        </div>
      </div>
    );
  }
}

export default withRouter(AnalyzerRedactorReportRow);
