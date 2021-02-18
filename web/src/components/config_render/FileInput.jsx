import React from "react";
import map from "lodash/map";
import after from "lodash/after";
import forEach from "lodash/forEach";

export default class FileInput extends React.Component {

  constructor(props) {
    super(props);
    this.state = {
      errText: "",
      fileAdded: false,
      fileName: "",
      fileNames: []
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
        this.props.onChange(
          map(files, "value"),
          map(files, "data")
        );
      }
    });

    forEach(ev.target.files, (file) => {
      var reader = new FileReader();
      reader.onload = () => {
        var vals = reader.result.split(",");
        if (vals.length !== 2) {
          error = "Invalid file data";
        } else {
          files.push({ value: file.name, data: vals[1] });
          if (this.props.multiple) {
            this.setState({ fileNames: files.map(file => file.value) })
          } else {
            this.setState({ fileName: files[0].value })
          }
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
          <p className="sub-header-color field-section-sub-header u-marginTop--15 u-marginBottom--small">{label}</p>
          <div className="flex flex-row">
            <div className={`${this.state.fileAdded || this.props.value ? "file-uploaded" : "custom-file-upload"}`}>
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
                <span className={`icon ${this.state.fileAdded || this.props.value ? "u-smallCheckGreen" : "u-ovalIcon clickable"} u-marginRight--10 u-top--3`}></span>
                {this.state.fileAdded || this.props.value ? this.props.multiple ? this.state.fileNames.join(",") : this.state.fileName : `Browse files for ${this.props.title}`}
                {this.state.fileAdded || this.props.value ? 
                  <p className="u-color--astral u-textDecoration--underlineOnHover u-fontSize--small u-marginLeft--30 u-marginTop--5">Select a different file</p>
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