import React from "react";
import Markdown from "react-remarkable";
import classNames from "classnames";
import { setOrder } from "./ConfigUtil";

export default class ConfigItemTitle extends React.Component {
  render() {
    const {
      title,
      recommended,
      required,
      hidden,
      when,
      error = "",
    } = this.props;

    var isHidden =
      hidden || when === "false" || (!title && !required && !recommended);

    if (isHidden) {
      return null;
    }

    return (
      <h4
        className="card-item-title field-section-sub-header"
        style={title?.length < 101 ? { marginBottom: -18 } : {}}
      >
        {title && (
          <div className="u-display--inlineBlock u-verticalAlign--top u-marginRight--5 break-word">
            <Markdown
              options={{
                linkTarget: "_blank",
                linkify: true,
              }}
            >
              {title}
            </Markdown>
          </div>
        )}
        <div
          className="u-display--inlineBlock u-verticalAlign--top"
          style={title?.length > 101 ? { marginTop: -12 } : {}}
        >
          {required ? (
            <span className="field-label required">Required</span>
          ) : recommended ? (
            <span className="field-label recommended">Recommended</span>
          ) : null}
          <span
            className={classNames("u-marginLeft--5 config-errblock", {
              visible: !!error,
            })}
            id={`${this.props.name}-errblock`}
          >
            {error || ""}
          </span>
        </div>
      </h4>
    );
  }
}
