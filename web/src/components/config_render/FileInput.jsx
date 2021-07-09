import React from "react";
import map from "lodash/map";
import after from "lodash/after";
import forEach from "lodash/forEach";

export default class FileInput extends React.Component {

  constructor(props) {
    super(props);
    this.state = {
      errText: "",
      fileAdded: false
    }
  }

  handleOnChange = (ev) => {
    this.setState({ errText: "" });

    let files = [];
    let error;

    const done = after(ev.target.files.length, () => {
      // this.refs.file.getDOMNode().value = "";
      if (error) {
        this.setState({ errText: error });
      } else if (this.props.onChange) {
        this.setState({ fileAdded: true })
        this.props.onChange(files);
      }
    });

    forEach(ev.target.files, (file) => {
      var reader = new FileReader();
      reader.onload = () => {
        var vals = reader.result.split(",");
        if (vals.length !== 2) {
          error = "Invalid file data";
        } else {
          files.push({ value: file.name, filename: vals[1] });
        }
        done();
      };
      reader.readAsDataURL(file);
    });
  }

  render() {
    let label;
    this.props.label ? label = this.props.label : this.props.multiple
      ? label = "Upload files" : label = "Upload a file";


    return (
      <div>
        <div className={`${this.props.readonly ? "readonly" : ""} ${this.props.disabled ? "disabled" : ""}`}>
          <p className="sub-header-color field-section-sub-header u-marginTop--10 u-marginBottom--5">{label}</p>
          <div className="flex flex-row">
            <div className={`${this.state.fileAdded || this.props.value || (this.props.multiple && this.props.filenamesText !== "") ? "file-uploaded" : "custom-file-upload"}`}>
              <input
                ref={(file) => this.file = file}
                type="file"
                name={this.props.name}
                className="inputfile"
                id={`${this.props.name} selector`}
                onChange={this.handleOnChange}
                readOnly={this.props.readOnly}
                multiple={this.props.multiple}
                disabled={this.props.disabled}
              />
              <label htmlFor={`${this.props.name} selector`} className="u-position--relative">
                <span className={`icon ${this.state.fileAdded || this.props.value || (this.props.multiple && this.props.filenamesText !== "") ? "u-smallCheckGreen" : "u-ovalIcon clickable"} u-marginRight--10 u-top--3`}></span>
                {this.state.fileAdded || this.props.value || (this.props.multiple && this.props.filenamesText !== "") ? this.props.filenamesText : `Browse files for ${this.props.title}`}
                {this.state.fileAdded || this.props.value || (this.props.multiple && this.props.filenamesText !== "") ? 
                  <p className="u-linkColor u-textDecoration--underlineOnHover u-fontSize--small u-marginLeft--30 u-marginTop--5">Select {this.props.multiple ? "other files" : "a different file"}</p>
                : null }
              </label>
            </div>
          </div>
        </div>
        <small className="text-danger"> {this.state.errText}</small>
      </div>
    );
  }
}
