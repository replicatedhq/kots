import React from "react";
import Markdown from "react-remarkable";

export default class ConfigCheckbox extends React.Component {

  handleOnChange = (e) => {
    const { handleOnChange, name } = this.props;
    var val = e.target.checked ? "1" : "0";
    if (this.props.handleOnChange && typeof handleOnChange === "function") {
      this.props.handleOnChange(name, val);
    }
  }

  render() {

    let val = this.props.value;
    if (!val || val.length === 0) {
      val = this.props.default;
    }
    var checked = val === "1";

    var hidden = this.props.hidden || this.props.when === "false";

    return (
      <div id={this.props.name} className={`field field-checkbox-wrapper u-marginTop--15 flex ${hidden ? "hidden" : ""}`}>
        <span className="u-marginTop--10 config-errblock" id={`${this.props.name}-errblock`}></span>
        <div className="flex1 flex u-marginRight--20">
          <input
            ref={(checkbox) => this.checkbox = checkbox}
            type="checkbox"
            name={this.props.name}
            id={this.props.name}
            value="1"
            checked={checked}
            readOnly={this.props.readonly}
            disabled={this.props.readonly}
            onChange={(e) => this.handleOnChange(e)}
            className={`${this.props.className || ""} flex-auto ${this.props.readonly ? "readonly" : ""}`} />
          <div>
            <label htmlFor={this.props.name} className={`u-marginLeft--10 header-color field-section-sub-header u-userSelect--none ${this.props.readonly ? "u-cursor--default" : "u-cursor--pointer"}`}>
              {this.props.title} {
                this.props.required ?
                  <span className="field-label required">Required</span> :
                  this.props.recommended ?
                    <span className="field-label recommended">Recommended</span> :
                    null}
            </label>
            {this.props.help_text !== "" ? 
              <div className="field-section-help-text u-marginTop--10">
                <Markdown
                  options={{
                    linkTarget: "_blank",
                    linkify: true,
                  }}>
                  {this.props.help_text}
                </Markdown>
              </div>
            : null}
          </div>
        </div>
      </div>
    );
  }
}
