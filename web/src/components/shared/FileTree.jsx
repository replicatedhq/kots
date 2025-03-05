import { Component } from "react";
import { rootPath } from "../../utilities/utilities";

export default class FileTree extends Component {
  state = {
    selected: {},
  };

  handleFileSelect = (path) => {
    this.props.handleFileSelect(path);
  };

  handleCheckboxChange = (filePath, isChecked) => {
    this.setState({
      selected: Object.assign({}, this.state.selected, {
        [filePath]: isChecked,
      }),
    });
  };

  getLevel = () => {
    return this.props.level || 0;
  };

  arePathsSame = (path1, path2) => {
    const newPath1 = rootPath(path1);
    const newPath2 = rootPath(path2);
    return (
      newPath1
        .split(/\//)
        .slice(1, 2 + this.getLevel())
        .join("/") ===
      newPath2
        .split(/\//)
        .slice(1, 2 + this.getLevel())
        .join("/")
    );
  };

  componentDidMount() {
    this.scrollToActiveFile("active-file");
  }

  scrollToActiveFile = (id) => {
    var e = document.getElementById(id);
    if (!!e && e.scrollIntoView) {
      e.scrollIntoView();
    }
  };

  render() {
    const { files, selectedFile, handleFileSelect, isRoot } = this.props;

    return (
      <ul className={`${isRoot ? "FileTree-wrapper" : "u-marginLeft--10"}`}>
        {files &&
          files.map((file, i) =>
            file.children && file.children.length ? (
              <li
                key={`${file.path}-Directory-${i}`}
                className="u-position--relative"
              >
                <input
                  type="checkbox"
                  data-testid={file.path}
                  checked={
                    this.state.selected.hasOwnProperty(file.path)
                      ? this.state.selected[file.path]
                      : this.arePathsSame(selectedFile, file.path)
                  }
                  onChange={(e) =>
                    this.handleCheckboxChange(file.path, e.target.checked)
                  }
                  name={`sub-dir-${file.name}-${file.children.length}-${
                    file.path
                  }-${i}-${this.getLevel()}`}
                  id={`sub-dir-${file.name}-${file.children.length}-${
                    file.path
                  }-${i}-${this.getLevel()}`}
                  data-testid={`support-bundle-analysis-file-tree-dir-${file.path}`}
                />
                <label
                  htmlFor={`sub-dir-${file.name}-${file.children.length}-${
                    file.path
                  }-${i}-${this.getLevel()}`}
                >
                  {file.name}
                </label>
                <FileTree
                  level={this.getLevel() + 1}
                  files={file.children}
                  handleFileSelect={(path) => handleFileSelect(path)}
                  selectedFile={selectedFile}
                />
              </li>
            ) : (
              <li
                key={`${file.path}-${i}`}
                id={`${selectedFile.includes(file.path) ? "active-file" : ""}`}
                title={file.name}
                className={`u-position--relative is-file ${
                  selectedFile.includes(file.path) ? "is-selected" : ""
                }`}
                data-testid={file.path}
                onClick={() => this.handleFileSelect(file.path)}
                data-testid={`support-bundle-analysis-file-tree-file-${file.path}`}
              >
                <div>{file.name}</div>
              </li>
            )
          )}
      </ul>
    );
  }
}
