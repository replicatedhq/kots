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

  renderFilesUploaded = (arr) => {
    if (!arr || arr.length === 0) { return null };
    return arr.map((item, index) => {
      return (
        <div key={`${item}-${index}`} className="u-marginTop--10" onClick={() => this.props.handleRemoveFile(this.props.name, item)}>
          <span className={`icon u-smallCheckGreen u-marginRight--10 u-top--3`}></span>
          {item}
          {arr.length > 1 ? <span className="icon red-trash-small clickable u-marginLeft--5 u-top--3" /> : null}
        </div>
      );
    });
  }

  render() {
    let label;
    this.props.label ? label = this.props.label : this.props.multiple
      ? label = "Upload files" : label = "Upload a file";
    const hasFileOrValue = this.state.fileAdded || this.props.value || (this.props.multiple && this.props.filenamesText !== "");

    return (
      <div>
        <div className={`${this.props.readonly ? "readonly" : ""} ${this.props.disabled ? "disabled" : ""}`}>
          <p className="sub-header-color field-section-sub-header u-marginTop--10 u-marginBottom--5">{label}</p>
          <div className="flex flex-row">
            <div className={`${hasFileOrValue ? "file-uploaded" : "custom-file-upload"}`}>
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
              {!this.props.multiple ?
                <label htmlFor={`${this.props.name} selector`} className="u-position--relative">
                  <span className={`icon ${hasFileOrValue ? "u-smallCheckGreen" : "u-ovalIcon clickable"} u-marginRight--10 u-top--3`}></span>
                  {hasFileOrValue ? this.props.filenamesText : `Browse files for ${this.props.title}`}
                  {hasFileOrValue ? 
                    <p className="u-linkColor u-textDecoration--underlineOnHover u-fontSize--small u-marginLeft--30 u-marginTop--5">Select a different file</p>
                  : null }
                </label>
              :
                <div>
                  {this.renderFilesUploaded(this.props.filenamesText)}
                  <label htmlFor={`${this.props.name} selector`} className="u-position--relative">
                    {hasFileOrValue ? 
                      <p className="u-linkColor u-textDecoration--underlineOnHover u-fontSize--small u-marginLeft--30 u-marginTop--10">Select other files</p>
                    : `Browse files for ${this.props.title}` }
                  </label>
                </div>
              }
            </div>
          </div>
        </div>
        <small className="text-danger"> {this.state.errText}</small>
      </div>
    );
  }
}
