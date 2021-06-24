import React from "react";
import FileInput from "./FileInput";
import ConfigItemTitle from "./ConfigItemTitle";

export default class ConfigFileInput extends React.Component {

  handleOnChange = (value, data) => {
    if (this.props.handleChange) {
      if (this.props.multiple) {
        this.props.handleChange(this.props.name, data, value);
      } else {
        this.props.handleChange(
          this.props.name,
          data ? data[0] : "",
          value ? value[0] : "",
        );
      }
    }
  }

  getFilenamesText = () => {
    if (this.props.multiple) {
      if (this.props.multi_data && this.props.multi_data.length) {
        return this.props.multi_data.join(", ");
      }
    } else if (this.props.data) {
      return this.props.data;
    } else if (this.props.filename) {
      return this.props.filename;
    }
    return this.props.default;
  }

  render() {
    var hidden = this.props.hidden || this.props.when === "false";
    
    return (
      <div id={`${this.props.name}-group`} className={`field field-type-file u-marginTop--15 ${hidden ? "hidden" : ""}`}>
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
                multiple={this.props.multiple}
                onChange={this.handleOnChange}
                filenamesText={this.getFilenamesText()}/>
            </span>
          </div>
        </div>
      </div>
    );
  }
}
