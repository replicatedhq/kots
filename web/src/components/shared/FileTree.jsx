import * as React from "react";
import { rootPath } from "../../utilities/utilities";

export default class FileTree extends React.Component {
  state = {
    selected: {},
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

  componentDidUpdate(lastProps) {
    const { isRoot, topLevelPaths, keepOpenPaths = [] } = this.props;
    const { initialOpenComplete } = this.state;

    if (isRoot && !initialOpenComplete && topLevelPaths && topLevelPaths !== lastProps.topLevelPaths) {
      const defaultSelected = topLevelPaths.reduce((current, path) => {
        let expand = true;
        if (keepOpenPaths?.length) {
          for (let i = 0; i < keepOpenPaths.length; i++) {
            const str = keepOpenPaths[i];
            expand = path.startsWith(str);
            if (expand) {
              break;
            }
          }
        }
        current[path] = expand;
        return current;
      }, {});

      let didInitialOpen = false;

      // The root folder(s) have already set themselves to be open.
      // Do not open root level folders anymore.
      if (Object.keys(defaultSelected).length) {
        didInitialOpen = true;
      }
      this.setState({
        selected: defaultSelected,
        initialOpenComplete: didInitialOpen
      });
    }
  }

  render() {
    const { files, selectedFile, handleFileSelect, isRoot } = this.props;
    return (
      <ul className={`${isRoot ? "FileTree-wrapper" : "u-marginLeft--10"}`}>
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
            <li key={`${file.path}-${i}`} title={file.name} className={`u-position--relative is-file ${selectedFile.includes(file.path) ? "is-selected" : ""}`} onClick={() => this.handleFileSelect(file.path)}>
              <div>{file.name}</div>
            </li>
        ))
        }
      </ul>
    );
  }
}
