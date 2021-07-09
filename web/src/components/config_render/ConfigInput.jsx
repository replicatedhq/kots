import React from "react";
import ConfigItemTitle from "./ConfigItemTitle";
import Markdown from "react-remarkable";

export default class ConfigInput extends React.Component {

  constructor(props) {
    super(props)
    this.inputRef = React.createRef();
    this.state = {
      inputVal: "",
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
      this.setState({ inputVal: this.props.value });
    }
  }

  componentDidMount() {
    if (this.props.value) {
      this.setState({ inputVal: this.props.value });
    }
    if (this.props.valuesByGroup) {
      Object.keys(this.props.valuesByGroup[this.props.groupName]).map((key) => {
        this.setState({
          [`${key}InputVal`]: this.props.valuesByGroup[this.props.groupName][key]
        })
      })
    }
  }

  maskValue = value => {
    if (!value) {
      return "";
    }
    return value.replace(/./g, '•');
  }

  // p1-2019-06-27
  // Fields that are required sometimes don't have a title associated with them.
  // Use title -OR- required prop to render <ConfigItemTitle> to make sure error
  // elements are rendered.
  render() {
    const hidden = this.props.hidden || this.props.when === "false";
    const placeholder = this.props.inputType === "password" ? this.maskValue(this.props.default) : "";  
    const isVariadic = this.props.valuesByGroup;
    const variadicItems = isVariadic ? Object.keys(this.props.valuesByGroup[this.props.groupName]) : {};
    const variadicItemsLen = variadicItems.length;
    return (
      isVariadic ? variadicItems.map((objKey, index) => {
        return (
          <div key={objKey} id={`${this.props.name}-group`} className={`field field-type-text ${hidden ? "hidden" : "u-marginTop--15"}`}>
            {this.props.title !== "" || this.props.required ?
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
            <div className="field-input-wrapper flex alignItems--center u-marginTop--15">
              <input
                ref={this.inputRef}
                type={this.props.inputType}
                {...this.props.props}
                placeholder={placeholder}
                value={this.state[`${objKey}InputVal`]}
                readOnly={this.props.readonly}
                disabled={this.props.readonly}
                onChange={(e) => this.handleOnChange(`${objKey}InputVal`, e, objKey)}
                onFocus={() => this.setState({ [`${objKey}InputFocused`]: true })}
                onBlur={() => this.setState({ [`${objKey}InputFocused`]: false })}
                className={`${this.props.className || ""} Input ${this.props.readonly ? "readonly" : ""}`} />
                {variadicItemsLen > 1 ?
                 <div className="icon gray-trash clickable u-marginLeft--10" onClick={() => this.props.handleRemoveItem(this.props.name, objKey)} />
                : null}
            </div>
            {this.props.inputType !== "password" && this.props.default ?
              <div className="default-value-section u-marginTop--8">
                Default value: <span className="value"> {this.props.default} </span>
              </div>
            : null}
            {variadicItemsLen === index + 1 &&
              <div className="u-marginTop--10" onClick={() => this.props.handleAddItem(this.props.name)}>
                <span className="add-btn u-fontSize--small u-fontWeight--bold u-linkColor u-cursor--pointer"><span className="icon u-addIcon--blue clickable" />Add another {this.props.title}</span>
              </div>
            }
          </div>
        );
      })
      :
      <div id={`${this.props.name}-group`} className={`field field-type-text ${hidden ? "hidden" : "u-marginTop--15"}`}>
        {this.props.title !== "" || this.props.required ?
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
        <div className="field-input-wrapper u-marginTop--15">
          <input
            ref={this.inputRef}
            type={this.props.inputType}
            {...this.props.props}
            placeholder={placeholder}
            value={this.state.inputVal}
            readOnly={this.props.readonly}
            disabled={this.props.readonly}
            onChange={(e) => this.handleOnChange("inputVal", e)}
            onFocus={() => this.setState({ focused: true })}
            onBlur={() => this.setState({ focused: false })}
            className={`${this.props.className || ""} Input ${this.props.readonly ? "readonly" : ""}`} />
        </div>
        {this.props.inputType !== "password" && this.props.default ?
          <div className="default-value-section u-marginTop--8">
            Default value: <span className="value"> {this.props.default} </span>
          </div>
        : null}
          {this.props.repeatable &&
            <div className="u-marginTop--10" onClick={() => this.props.handleAddItem(this.props.name)}>
              <span className="add-btn u-fontSize--small u-fontWeight--bold u-linkColor u-cursor--pointer"><span className="icon u-addIcon--blue clickable" />Add another {this.props.title}</span>
            </div>
          }
      </div>
    );
  }
}
