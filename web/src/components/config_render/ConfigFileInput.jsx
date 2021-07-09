import React from "react";
import FileInput from "./FileInput";
import ConfigItemTitle from "./ConfigItemTitle";
import map from "lodash/map";
export default class ConfigFileInput extends React.Component {

  handleOnChange = (files) => {
    if (this.props.handleChange) {
      if (this.props.repeatable) {
        this.props.handleChange(this.props.name, files, "");
      } else {
        const data = map(files.filter(f => f), "filename");
        const value = map(files.filter(f => f), "value");
        // TODO: @GraysonNull (07/09/2021) This is backwards but switching it breaks things and I don't have the time to search through and fix it all right now.
        this.props.handleChange(
          this.props.name,
          data ? data[0] : "",
          value ? value[0] : "",
        );
      }
    }
  }

  getFilenamesText = () => {
    if (this.props.repeatable) {
      if (this.props.valuesByGroup && Object.keys(this.props.valuesByGroup[this.props.groupName]).length) {
        const filenames= [];
        Object.keys(this.props.valuesByGroup[this.props.groupName]).map((key) => {
          filenames.push(key);
        });
        if (filenames.length > 0) {
          return filenames.join(", ");
        }
      }
    } else if (this.props.data) {
      return this.props.data;
    } else if (this.props.filename) {
      return this.props.filename;
    } else {
      return this.props.default;
    }
  }

  render() {
    var hidden = this.props.hidden || this.props.when === "false";
    
    return (
      <div id={`${this.props.name}-group`} className={`field-type-file ${hidden ? "hidden" : ""}`}>
        {this.props.title !== "" ?
          <ConfigItemTitle
            title={this.props.title}
            recommended={this.props.recommended}
            required={this.props.required}
            name={this.props.name}
            error={this.props.error}
          />
          : null}
        <div className="input input-type-file clearfix">
          <div>
            <span>
              <FileInput
                ref={(file) => this.file = file}
                name={this.props.name}
                title={this.props.title}
                value={this.props.value}
                readOnly={this.props.readonly}
                disabled={this.props.readonly}
                multiple={this.props.repeatable}
                onChange={this.handleOnChange}
                filenamesText={this.getFilenamesText()}/>
            </span>
          </div>
        </div>
      </div>
    );
  }
}
