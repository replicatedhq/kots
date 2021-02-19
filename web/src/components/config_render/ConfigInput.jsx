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

  handleOnChange = (e) => {
    const { handleOnChange, name } = this.props;
    this.setState({ inputVal: e.target.value });
    if (handleOnChange && typeof handleOnChange === "function") {
      handleOnChange(name, e.target.value);
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
    var hidden = this.props.hidden || this.props.when === "false";
    var placeholder = this.props.inputType === "password" ? this.maskValue(this.props.default) : this.props.default;

    return (
      <div id={this.props.name} className={`field field-type-text ${hidden ? "hidden" : "u-marginTop--15"}`}>
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
        <div className="field-input-wrapper u-marginTop--15">
          <input
            ref={this.inputRef}
            type={this.props.inputType}
            {...this.props.props}
            placeholder={placeholder}
            value={this.state.inputVal}
            readOnly={this.props.readonly}
            disabled={this.props.readonly}
            onChange={(e) => this.handleOnChange(e)}
            onFocus={() => this.setState({ focused: true })}
            onBlur={() => this.setState({ focused: false })}
            className={`${this.props.className || ""} Input ${this.props.readonly ? "readonly" : ""}`} />
        </div>
      </div>
    );
  }
}
