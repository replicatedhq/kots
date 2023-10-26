import { Component, createRef } from "react";
import ConfigItemTitle from "./ConfigItemTitle";
import Markdown from "react-remarkable";
import { setOrder } from "./ConfigUtil";
import { ConfigWrapper } from "./ConfigComponents";

export default class ConfigTextarea extends Component {
  constructor(props) {
    super(props);
    this.textareaRef = createRef();
    this.state = {
      textareaVal: "",
      focused: false,
    };
  }

  handleOnChange = (field, e, objKey) => {
    const { handleOnChange, name } = this.props;
    this.setState({ [`${field}`]: e.target.value });
    if (handleOnChange && typeof handleOnChange === "function") {
      handleOnChange(name, e.target.value, objKey);
    }
  };

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
          [`${key}TextareaVal`]:
            this.props.valuesByGroup[this.props.groupName][key],
        });
      });
    }
  }

  render() {
    var hidden = this.props.hidden || this.props.when === "false";
    const isVariadic = this.props.valuesByGroup;
    const variadicItems = isVariadic
      ? Object.keys(this.props.valuesByGroup[this.props.groupName])
      : {};
    const variadicItemsLen = variadicItems.length;
    return isVariadic ? (
      variadicItems.map((objKey, index) => {
        return (
          <ConfigWrapper
            key={objKey}
            id={`${this.props.name}-group`}
            className={`field-type-text`}
            marginTop={hidden || this.props.affix ? "0" : "35px"}
            hidden={hidden}
            order={setOrder(this.props.index, this.props.affix)}
          >
            {this.props.title !== "" ? (
              <ConfigItemTitle
                title={this.props.title}
                recommended={this.props.recommended}
                required={this.props.required}
                name={this.props.name}
                error={this.props.error}
              />
            ) : null}
            {this.props.help_text !== "" ? (
              <div className="field-section-help-text help-text-color">
                <Markdown
                  options={{
                    linkTarget: "_blank",
                    linkify: true,
                  }}
                >
                  {this.props.help_text}
                </Markdown>
              </div>
            ) : null}
            <div className="field-input-wrapper flex alignItems--center u-marginTop--10">
              <textarea
                ref={this.textareaRef}
                {...this.props.props}
                placeholder={this.props.default}
                value={this.state[`${objKey}TextareaVal`]}
                readOnly={this.props.readonly}
                disabled={this.props.readonly}
                onChange={(e) =>
                  this.handleOnChange(`${objKey}TextareaVal`, e, objKey)
                }
                onFocus={() =>
                  this.setState({ [`${objKey}TextareaFocused`]: true })
                }
                onBlur={() =>
                  this.setState({ [`${objKey}TextareaFocused`]: false })
                }
                className={`${this.props.className || ""} Textarea ${
                  this.props.readonly ? "readonly" : ""
                }`}
              ></textarea>
              {variadicItemsLen > 1 ? (
                <Icon
                  icon="trash"
                  size={20}
                  className="gray-color u-marginLeft--10 clickable"
                  onClick={() =>
                    this.props.handleRemoveItem(this.props.name, objKey)
                  }
                />
              ) : null}
            </div>
            {variadicItemsLen === index + 1 && (
              <div
                className="u-marginTop--10"
                onClick={() => this.props.handleAddItem(this.props.name)}
              >
                <span className="add-btn u-fontSize--small u-fontWeight--bold link">
                  <Icon icon="plus" size={14} className="clickable" />
                  Add another {this.props.title}
                </span>
              </div>
            )}
          </ConfigWrapper>
        );
      })
    ) : (
      <ConfigWrapper
        id={`${this.props.name}-group`}
        className={`field-type-text`}
        marginTop={hidden || this.props.affix ? "0" : "35px"}
        hidden={hidden}
        order={setOrder(this.props.index, this.props.affix)}
      >
        {this.props.title !== "" ? (
          <ConfigItemTitle
            title={this.props.title}
            recommended={this.props.recommended}
            required={this.props.required}
            name={this.props.name}
            error={this.props.error}
          />
        ) : null}
        {this.props.help_text !== "" ? (
          <div className="field-section-help-text help-text-color">
            <Markdown
              options={{
                linkTarget: "_blank",
                linkify: true,
              }}
            >
              {this.props.help_text}
            </Markdown>
          </div>
        ) : null}
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
            className={`${this.props.className || ""} Textarea ${
              this.props.readonly ? "readonly" : ""
            } ${this.props.showValidationError ? "has-error" : ""}`}
          ></textarea>
          {this.props.showValidationError && (
            <div className="config-input-error-message tw-mt-1 tw-text-xs">
              {this.props.validationErrorMessage}
            </div>
          )}
        </div>
        {this.props.repeatable && (
          <div
            className="u-marginTop--10"
            onClick={() => this.props.handleAddItem(this.props.name)}
          >
            <span className="add-btn u-fontSize--small u-fontWeight--bold link">
              <Icon icon="plus" size={14} className="clickable" />
              Add another {this.props.title}
            </span>
          </div>
        )}
      </ConfigWrapper>
    );
  }
}
