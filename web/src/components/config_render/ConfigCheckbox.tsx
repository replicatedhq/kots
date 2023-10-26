import { ChangeEvent, Component, createRef } from "react";
// TODO: add type checking support for react-remarkable or add a global ignore
// @ts-ignore
import Markdown from "react-remarkable";
import { setOrder } from "./ConfigUtil";
import { ConfigWrapper } from "./ConfigComponents";

type Props = {
  default: string;
  groupName: string;
  handleAddItem: () => void;
  handleOnChange: (name: String, val: string) => void;
  handleRemoveItem: () => void;
  help_text: string;
  hidden: boolean;
  index: number;
  name: string;
  readonly: boolean;
  title: string;
  type: string;
  value: string;
  when: string;
  affix: string;
  className: string;
  required: boolean;
  recommended: boolean;
};

export default class ConfigCheckbox extends Component<Props> {
  private checkbox = createRef<HTMLInputElement>();

  handleOnChange = (e: ChangeEvent<HTMLInputElement>): void => {
    const { handleOnChange, name } = this.props;
    var val = e.target.checked ? "1" : "0";
    if (this.props.handleOnChange && typeof handleOnChange === "function") {
      this.props.handleOnChange(name, val);
    }
  };

  render() {
    let val = this.props.value;
    if (!val || val.length === 0) {
      val = this.props.default;
    }
    var checked = val === "1";

    var hidden = this.props.hidden || this.props.when === "false";

    return (
      <ConfigWrapper
        id={`${this.props.name}-group`}
        className={`field-checkbox-wrapper`}
        marginTop={hidden || this.props.affix ? "0" : "15px"}
        hidden={hidden}
        order={setOrder(this.props.index, this.props.affix)}
      >
        <span
          className="u-marginTop--10 config-errblock"
          id={`${this.props.name}-errblock`}
        ></span>
        <div className="flex1 flex u-marginRight--20">
          <input
            ref={this.checkbox}
            type="checkbox"
            name={this.props.name}
            id={this.props.name}
            value="1"
            checked={checked}
            readOnly={this.props.readonly}
            disabled={this.props.readonly}
            onChange={(e) => this.handleOnChange(e)}
            className={`${this.props.className || ""} flex-auto ${
              this.props.readonly ? "readonly" : ""
            }`}
          />
          <label
            htmlFor={this.props.name}
            className={`u-marginLeft--5 field-section-sub-header card-item-title u-userSelect--none ${
              this.props.readonly ? "u-cursor--default" : "u-cursor--pointer"
            }`}
          >
            {this.props.title}{" "}
            {this.props.required ? (
              <span className="field-label required">Required</span>
            ) : this.props.recommended ? (
              <span className="field-label recommended">Recommended</span>
            ) : null}
          </label>
        </div>
        {this.props.help_text !== "" ? (
          <div
            className="field-section-help-text help-text-color"
            style={{ marginLeft: "25px" }}
          >
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
      </ConfigWrapper>
    );
  }
}
