import * as React from "react";
import { rootPath } from "../../utilities/utilities";

export default class FileTree extends React.Component {
  constructor() {
    super();
    this.state = {
      selected: {}
    }
  }

  handleFileSelect = (path) => {
    this.props.handleFileSelect(path);
  }

  handleCheckboxChange = (filePath, isChecked) => {
    this.setState({
      selected: Object.assign({}, this.state.selected, {
        [filePath]: isChecked
      })
    })
  }

  getLevel = () => {
    return this.props.level || 0
  }

  arePathsSame = (path1, path2) => {
    const newPath1 = rootPath(path1);
    const newPath2 = rootPath(path2);
    return newPath1.split(/\//).slice(1, 2+this.getLevel()).join("/") === newPath2.split(/\//).slice(1, 2+this.getLevel()).join("/")
  }

  render() {
    const { files, selectedFile, handleFileSelect } = this.props;

    return (
      <ul className={`${this.props.isRoot ? "FileTree-wrapper" : "u-marginLeft--normal"}`}>
        {files && files.map((file, i) => (
          file.children && file.children.length ?
            <li key={`${file.path}-Directory-${i}`} className="u-position--relative">
              <input type="checkbox"
                checked={this.state.selected.hasOwnProperty(file.path) ? this.state.selected[file.path] : this.arePathsSame(selectedFile, file.path)}
                onChange={e => this.handleCheckboxChange(file.path, e.target.checked)}
                name={`sub-dir-${file.name}-${file.children.length}-${file.path}-${i}`}
                id={`sub-dir-${file.name}-${file.children.length}-${file.path}-${i}`} />
              <label htmlFor={`sub-dir-${file.name}-${file.children.length}-${file.path}-${i}`}>{file.name}</label>
              <FileTree
                level={this.getLevel() + 1}
                files={file.children}
                handleFileSelect={(path) => handleFileSelect(path)}
                selectedFile={selectedFile}
              />
            </li>
            :
            <li key={file.path} className={`u-position--relative is-file ${selectedFile === file.path ? "is-selected" : ""}`} onClick={() => this.handleFileSelect(file.path)}>{file.name}</li>
        ))
        }
      </ul>
    );
  }
}
