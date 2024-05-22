import { Component } from "react";
import FileInput from "./FileInput";
import ConfigItemTitle from "./ConfigItemTitle";
import map from "lodash/map";
import Markdown from "react-remarkable";

export default class ConfigFileInput extends Component {
  handleOnChange = (files) => {
    if (this.props.handleChange) {
      if (this.props.repeatable) {
        this.props.handleChange(this.props.name, files, "");
      } else {
        const data = map(
          files.filter((f) => f),
          "filename"
        );
        const value = map(
          files.filter((f) => f),
          "value"
        );
        // TODO: @GraysonNull (07/09/2021) This is backwards but switching it breaks things and I don't have the time to search through and fix it all right now.
        this.props.handleChange(
          this.props.name,
          data ? data[0] : "",
          value ? value[0] : ""
        );
      }
    }
  };

  handleDownloadFile = async (fileName) => {
    // TODO NOW: download from upgrader if rendered in upgrader and use a different sequence!
    const url = `${process.env.API_ENDPOINT}/app/${this.props.appSlug}/config/${this.props.configSequence}/${fileName}/download`;
    fetch(url, {
      method: "GET",
      headers: {
        "Content-Type": "application/octet-stream",
      },
      credentials: "include",
    })
      .then((response) => {
        if (!response.ok) {
          throw Error(response.statusText); // TODO: handle error
        }
        return response.blob();
      })
      .then((blob) => {
        const downloadURL = window.URL.createObjectURL(new Blob([blob]));
        const link = document.createElement("a");
        link.href = downloadURL;
        link.setAttribute("download", fileName);
        document.body.appendChild(link);
        link.click();
        link.parentNode.removeChild(link);
      })
      .catch(function (error) {
        console.log(error); // TODO handle error
      });
  };

  getFilenamesText = () => {
    if (this.props.repeatable) {
      if (
        this.props.valuesByGroup &&
        Object.keys(this.props.valuesByGroup[this.props.groupName]).length
      ) {
        const filenames = [];
        Object.keys(this.props.valuesByGroup[this.props.groupName]).map(
          (key) => {
            filenames.push(key);
          }
        );
        if (filenames.length > 0) {
          return filenames;
        }
      }
    } else if (this.props.data) {
      return this.props.data;
    } else if (this.props.filename) {
      return this.props.filename;
    } else {
      return this.props.default;
    }
  };

  render() {
    var hidden = this.props.hidden || this.props.when === "false";
    return (
      <div
        id={`${this.props.name}-group`}
        className={`field-type-file ${hidden ? "hidden" : ""}`}
      >
        {this.props.title !== "" ? (
          <ConfigItemTitle
            title={this.props.title}
            recommended={this.props.recommended}
            required={this.props.required}
            name={this.props.name}
            error={this.props.error}
          />
        ) : null}
        <div className="field-input-wrapper input input-type-file clearfix">
          <div>
            {this.props.help_text !== "" ? (
              <div className="field-section-help-text help-text-color">
                <Markdown
                  options={{
                    linkTarget: "_blank",
                    linkify: true,
                  }}
                >
                  {this.props.help_text}
                </Markdown>
              </div>
            ) : null}
            <span>
              <FileInput
                ref={(file) => (this.file = file)}
                name={this.props.name}
                title={this.props.title}
                value={this.props.value}
                readOnly={this.props.readonly}
                disabled={this.props.readonly}
                multiple={this.props.repeatable}
                onChange={this.handleOnChange}
                filenamesText={this.getFilenamesText()}
                handleRemoveFile={(itemName, itemToRemove) =>
                  this.props.handleRemoveItem(itemName, itemToRemove)
                }
                handleDownloadFile={(fileName) =>
                  this.handleDownloadFile(fileName)
                }
              />
            </span>
            {this.props.showValidationError && (
              <div className="config-input-error-message tw-mt-1 tw-text-xs">
                {this.props.validationErrorMessage}
              </div>
            )}
          </div>
        </div>
      </div>
    );
  }
}
