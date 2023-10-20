import { Component } from "react";
import { DiffEditor as MonacoDiffEditor } from "@monaco-editor/react";

import { diffContent } from "../../utilities/utilities";

export default class DiffEditor extends Component {
  state = {
    addedLines: 0,
    removedLines: 0,
    changes: 0,
  };

  componentDidMount() {
    const lineChanges = diffContent(
      this.props.original || "",
      this.props.value || ""
    );
    this.setState(lineChanges);
  }

  render() {
    const { addedLines, removedLines, changes } = this.state;
    const { original, value, specKey } = this.props;

    return (
      <div className="flex flex1 flex-column">
        <div className="flex alignItems--center">
          {addedLines || removedLines || changes ? (
            <div className="flex u-marginRight--10">
              <span className="u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-textColor--success u-marginRight--5">
                {" "}
                {`+${addedLines} ${addedLines >= 0 ? "additions" : "addition"}`}
              </span>
              <span className="u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-textColor--error u-marginRight--5">
                {" "}
                {`-${removedLines} ${
                  removedLines >= 0 ? "subtractions" : "subtraction"
                }`}
              </span>
              <span className="u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-textColor--accent">
                {" "}
                {`${changes} ${changes > 1 ? "changes" : "change"}`}
              </span>
            </div>
          ) : null}
          {specKey}
        </div>
        <div className="MonacoDiffEditor--wrapper flex flex1 u-height--full u-width--full u-marginTop--5 u-marginBottom--20">
          <div className="flex-column u-width--full u-overflow--hidden">
            <div className="flex-column flex flex1">
              <MonacoDiffEditor
                ref={(editor) => {
                  this.monacoDiffEditor = editor;
                }}
                width="100%"
                height="100%"
                language="yaml"
                original={original || ""}
                modified={value || ""}
                onChange={this.onEditorValuesLoaded}
                options={{
                  enableSplitViewResizing: true,
                  scrollBeyondLastLine: false,
                  readOnly: true,
                }}
              />
            </div>
          </div>
        </div>
      </div>
    );
  }
}
