import React from "react";
import ConfigItemTitle from "./ConfigItemTitle";
import Markdown from "react-remarkable";

export default class ConfigTextarea extends React.Component {

  constructor(props) {
    super(props)
    this.textareaRef = React.createRef();
    this.state = {
      textareaVal: "",
      focused: false
    }
  }

  handleOnChange = (field, e, objKey) => {
    const { handleOnChange, name } = this.props;
    this.setState({ [`${field}`]: e.target.value });
    if (handleOnChange && typeof handleOnChange === "function") {
      handleOnChange(name, e.target.value, objKey);
    }
  }

  componentDidUpdate(lastProps) {
    if (this.props.value !== lastProps.value && !this.state.focused) {
      this.setState({ textareaVal: this.props.value });
    }
  }

  componentDidMount() {
    if (this.props.value) {
      this.setState({ textareaVal: this.props.value });
    }
    if (this.props.valuesByGroup) {
      Object.keys(this.props.valuesByGroup[this.props.groupName]).map((key) => {
        this.setState({
          [`${key}TextareaVal`]: this.props.valuesByGroup[this.props.groupName][key]
        })
      })
    }
  }

  render() {
    var hidden = this.props.hidden || this.props.when === "false";
    const isVariadic = this.props.valuesByGroup;
    const variadicItems = isVariadic ? Object.keys(this.props.valuesByGroup[this.props.groupName]) : {};
    const variadicItemsLen = variadicItems.length;
    return (
      isVariadic ? variadicItems.map((objKey, index) => {
        return (
          <div key={objKey} id={`${this.props.name}-group`} className={`field field-type-text u-marginTop--15 ${hidden ? "hidden" : ""}`}>
            {this.props.title !== "" ?
              <ConfigItemTitle
                title={this.props.title}
                recommended={this.props.recommended}
                required={this.props.required}
                name={this.props.name}
                error={this.props.error}
              />
              : null}
            {this.props.help_text !== "" ?
              <div className="field-section-help-text u-marginTop--5">
                <Markdown
                  options={{
                    linkTarget: "_blank",
                    linkify: true,
                  }}>
                  {this.props.help_text}
                </Markdown>
              </div>
            : null}
            <div className="field-input-wrapper flex alignItems--center u-marginTop--10">
              <textarea
                ref={this.textareaRef}
                {...this.props.props}
                placeholder={this.props.default}
                value={this.state[`${objKey}TextareaVal`]}
                readOnly={this.props.readonly}
                disabled={this.props.readonly}
                onChange={(e) => this.handleOnChange(`${objKey}TextareaVal`, e, objKey)}
                onFocus={() => this.setState({ [`${objKey}TextareaFocused`]: true })}
                onBlur={() => this.setState({ [`${objKey}TextareaFocused`]: false })}
                className={`${this.props.className || ""} Textarea ${this.props.readonly ? "readonly" : ""}`}>
              </textarea>
              {variadicItemsLen > 1 ?
                <div className="icon gray-trash clickable u-marginLeft--10" onClick={() => this.props.handleRemoveItem(this.props.name, objKey)} />
              : null}
            </div>
            {variadicItemsLen === index + 1 &&
              <div className="u-marginTop--10" onClick={() => this.props.handleAddItem(this.props.name)}>
                <span className="add-btn u-fontSize--small u-fontWeight--bold u-linkColor u-cursor--pointer"><span className="icon u-addIcon--blue clickable" />Add another {this.props.title}</span>
              </div>
            }
          </div>
        )
      }) :
        <div id={`${this.props.name}-group`} className={`field field-type-text u-marginTop--15 ${hidden ? "hidden" : ""}`}>
          {this.props.title !== "" ?
            <ConfigItemTitle
              title={this.props.title}
              recommended={this.props.recommended}
              required={this.props.required}
              name={this.props.name}
              error={this.props.error}
            />
            : null}
          {this.props.help_text !== "" ?
            <div className="field-section-help-text u-marginTop--5">
              <Markdown
                options={{
                  linkTarget: "_blank",
                  linkify: true,
                }}>
                {this.props.help_text}
              </Markdown>
            </div>
          : null}
          <div className="field-input-wrapper u-marginTop--10">
            <textarea
              ref={this.textareaRef}
              {...this.props.props}
              placeholder={this.props.default}
              value={this.state.textareaVal}
              readOnly={this.props.readonly}
              disabled={this.props.readonly}
              onChange={(e) => this.handleOnChange("textareaVal", e)}
              onFocus={() => this.setState({ focused: true })}
              onBlur={() => this.setState({ focused: false })}
              className={`${this.props.className || ""} Textarea ${this.props.readonly ? "readonly" : ""}`}>
            </textarea>
          </div>
          {this.props.repeatable &&
            <div className="u-marginTop--10" onClick={() => this.props.handleAddItem(this.props.name)}>
              <span className="add-btn u-fontSize--small u-fontWeight--bold u-linkColor u-cursor--pointer"><span className="icon u-addIcon--blue clickable" />Add another {this.props.title}</span>
            </div>
          }
        </div>
    );
  }
}
