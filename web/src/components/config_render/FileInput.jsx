import { Component } from "react";
import after from "lodash/after";
import forEach from "lodash/forEach";
import Icon from "../Icon";

export default class FileInput extends Component {
  constructor(props) {
    super(props);
    this.state = {
      errText: "",
      fileAdded: false,
      showDownloadIcon: true,
    };
  }

  handleRemoveFile = (name, item) => {
    if (!item) {
      // single file remove
      this.props.onChange([{ filename: "", value: "" }]);
      this.setState({ fileAdded: false });
    } else {
      // variadic config item remove
      this.props.handleRemoveFile(name, item);
    }
  };

  handleDownloadFile = () => {
    this.props.handleDownloadFile(this.props.filenamesText);
  };

  handleOnChange = (ev) => {
    this.setState({ errText: "" });

    let files = [];
    let error;

    const done = after(ev.target.files.length, () => {
      // this.refs.file.getDOMNode().value = "";
      if (error) {
        this.setState({ errText: error });
      } else if (this.props.onChange) {
        this.setState({ fileAdded: true, showDownloadIcon: false });
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
  };

  renderFilesUploaded = (arr) => {
    if (!arr || arr.length === 0) {
      return null;
    }
    return arr.map((item, index) => {
      return (
        <div key={`${item}-${index}`} className="u-marginTop--10">
          <Icon
            icon="check-circle-filled"
            size={18}
            className="success-color u-marginRight--10 u-top--3"
          />
          {item}
          {arr.length > 1 ? (
            <Icon
              icon="trash"
              size={16}
              className="gray-color clickable u-marginLeft--5 u-top--3"
              onClick={() => this.handleRemoveFile(this.props.name, item)}
            />
          ) : null}
        </div>
      );
    });
  };

  render() {
    let label;
    this.props.label
      ? (label = this.props.label)
      : this.props.multiple
      ? (label = "Upload files")
      : (label = "Upload a file");
    const hasFileOrValue =
      this.state.fileAdded ||
      this.props.value ||
      (this.props.multiple && this.props.filenamesText !== "");

    return (
      <div>
        <div
          className={`${this.props.readonly ? "readonly" : ""} ${
            this.props.disabled ? "disabled" : ""
          }`}
        >
          <p className="card-item-title field-section-sub-header u-marginTop--10 u-marginBottom--5">
            {label}
          </p>
          <div className="flex flex-row">
            <div
              className={`${
                hasFileOrValue ? "file-uploaded" : "custom-file-upload"
              }`}
            >
              <input
                ref={(file) => (this.file = file)}
                type="file"
                name={this.props.name}
                className="inputfile"
                id={`${this.props.name} selector`}
                onChange={this.handleOnChange}
                readOnly={this.props.readOnly}
                multiple={this.props.multiple}
                disabled={this.props.disabled}
              />
              {!this.props.multiple ? (
                hasFileOrValue ? (
                  <div>
                    <div>
                      <Icon
                        icon="check-circle-filled"
                        size={18}
                        className="clickable success-color u-marginRight--10 u-top--3"
                      />
                      {this.props.filenamesText}
                      <Icon
                        icon="trash"
                        size={16}
                        className="clickable gray-color u-marginLeft--10 u-top--3"
                        onClick={() => this.handleRemoveFile(this.props.name)}
                      />
                      {this.state.showDownloadIcon && (
                        <Icon
                          icon="download"
                          size={16}
                          className="clickable gray-color u-marginLeft--10 u-top--3"
                          onClick={this.handleDownloadFile}
                        />
                      )}
                    </div>
                    <label
                      htmlFor={`${this.props.name} selector`}
                      className="u-position--relative"
                    >
                      <p className="link u-textDecoration--underlineOnHover u-fontSize--small u-marginLeft--30 u-marginTop--5">
                        Select a different file
                      </p>
                    </label>
                  </div>
                ) : (
                  <label
                    htmlFor={`${this.props.name} selector`}
                    className="u-position--relative"
                  >
                    <Icon
                      icon="dotted-circle"
                      size={16}
                      className="clickable gray-color u-marginRight--10 u-top--3"
                    />
                    Browse files for {this.props.title}
                  </label>
                )
              ) : (
                <div>
                  {this.renderFilesUploaded(this.props.filenamesText)}
                  <label
                    htmlFor={`${this.props.name} selector`}
                    className="u-position--relative"
                  >
                    {hasFileOrValue ? (
                      <p className="link u-textDecoration--underlineOnHover u-fontSize--small u-marginLeft--30 u-marginTop--10">
                        Select other files
                      </p>
                    ) : (
                      `Browse files for ${this.props.title}`
                    )}
                  </label>
                </div>
              )}
            </div>
          </div>
        </div>
        <small className="text-danger"> {this.state.errText}</small>
      </div>
    );
  }
}
