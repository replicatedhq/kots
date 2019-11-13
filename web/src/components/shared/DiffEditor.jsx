import * as React from "react";
import { MonacoDiffEditor } from "react-monaco-editor";

import { getLineChanges } from "../../utilities/utilities";

export default class DiffEditor extends React.Component {
  state = {
    addedLines: 0,
    removedLines: 0,
    changes: 0
  }

  onEditorValuesLoaded = () => {
    if (this.monacoDiffEditor) {
      const lineChanges = getLineChanges(this.monacoDiffEditor.editor.getLineChanges());
      this.setState(lineChanges)
    }
  }

  render() {
    const { addedLines, removedLines, changes } = this.state;
    const { original, value, specKey } = this.props;

    return (
      <div className="flex flex1 flex-column">
        <div className="flex alignItems--center">
          {addedLines || removedLines || changes ?
            <div className="flex u-marginRight--10">
              <span className="u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-color--vidaLoca u-marginRight--5"> {`+${addedLines} ${addedLines >= 0 ? "additions" : "addition"}`}</span>
              <span className="u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-color--chestnut u-marginRight--5"> {`-${removedLines} ${removedLines >= 0 ? "subtractions" : "subtraction"}`}</span>
              <span className="u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-color--doveGray"> {`${changes} ${changes > 1 ? "changes" : "change"}`}</span>
            </div>
            : null}
          {specKey}
        </div>
          <div className="MonacoDiffEditor--wrapper flex flex1 u-height--full u-width--full u-marginTop--5 u-marginBottom--20">
            <div className="flex-column u-width--full u-overflow--hidden">
              <div className="flex-column flex flex1">
                <MonacoDiffEditor
                  ref={(editor) => { this.monacoDiffEditor = editor }}
                  width="100%"
                  height="100%"
                  language="yaml"
                  original={original?.content || ""}
                  value={value?.content || ""}
                  onChange={this.onEditorValuesLoaded}
                  options={{
                    enableSplitViewResizing: true,
                    scrollBeyondLastLine: false,
                    readOnly: true
                  }}
                />
              </div>
            </div>
          </div>
      </div>
    );
  }
}
