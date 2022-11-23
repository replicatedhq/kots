import React from 'react';
import get from 'lodash/get';
import { RadioButton } from 'replicated-design-system';

export default class ConfigRadio extends React.Component {
  handleOnChange = (e) => {
    const { group } = this.props;
    if (
      this.props.handleChange &&
      typeof this.props.handleChange === 'function'
    ) {
      this.props.handleChange(group, e.target.value);
    }
  };

  render() {
    let val = get(this.props, 'value');
    if (!val || val.length === 0) {
      val = this.props.default;
    }
    const checked = val === this.props.name;

    return (
      <RadioButton
        name={this.props.name}
        divClassName="u-marginRight--20 u-marginTop--15"
        group={this.props.group}
        checked={checked}
        title={this.props.title}
        disabled={this.props.readOnly}
        inputClassName={this.props.className}
        labelClassName="u-marginLeft--5 header-color field-section-sub-header u-userSelect--none"
        onChange={this.handleOnChange}
      />
    );
  }
}
