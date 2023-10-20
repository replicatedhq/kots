import { Component } from "react";
import Icon from "@components/Icon";
import { Link } from "react-router-dom";

import { Utilities } from "../../utilities/utilities";

class RedactorRow extends Component {
  state = {
    redactorEnabled: false,
  };

  handleEnableRedactor = () => {
    this.setState({ redactorEnabled: !this.state.redactorEnabled }, () => {
      this.props.handleSetRedactEnabled(
        this.props.redactor,
        this.state.redactorEnabled
      );
    });
  };

  componentDidMount() {
    if (this.props.redactor) {
      this.setState({
        redactorEnabled: this.props.redactor.enabled ? true : false,
      });
    }
  }

  handleDeleteClick = (redactor) => {
    this.props.toggleConfirmDeleteModal(redactor);
  };

  render() {
    const { redactor } = this.props;

    return (
      <div
        className="flex flex-auto RedactorRowDetails card-item"
        key={redactor?.slug}
      >
        <div className="flex-column flex1">
          <div className="flex flex1 alignItems--center u-marginBottom--small">
            <span
              className={`status-indicator u-marginBottom--10 ${
                this.state.redactorEnabled ? "enabled" : "disabled"
              }`}
            />
            <p className="u-fontSize--large u-lineHeight--normal u-fontWeight--bold card-item-title">
              {redactor?.name}
            </p>
          </div>
          <span className="u-fontSize--small u-textColor--bodyCopy u-fontWeight--medium u-lineHeight--normal u-marginTop--5">
            {" "}
            Last updated on{" "}
            {Utilities.dateFormat(
              redactor?.updatedAt,
              "MM/DD/YY @ hh:mm a z"
            )}{" "}
          </span>
          <p className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-textColor--accent u-marginLeft--10">
            {" "}
            {redactor?.description}{" "}
          </p>
        </div>
        <div className="flex alignItems--center">
          <Link
            to={`/app/${this.props.appSlug}/troubleshoot/redactors/${redactor?.slug}`}
            className="u-fontSize--normal link u-textDecoration--underlineOnHover u-marginRight--20"
          >
            <Icon icon="edit" size={18} className="clickable" />
          </Link>
          <span
            className="u-fontSize--normal u-fontWeight--medium u-textColor--error u-textDecoration--underlineOnHover u-marginRight--20"
            onClick={() => this.handleDeleteClick(redactor)}
          >
            <Icon icon="trash" size={18} className="clickable error-color" />
          </span>
          <div
            className={`Checkbox--switch ${
              this.state.redactorEnabled ? "is-checked" : "is-notChecked"
            }`}
          >
            <input
              type="checkbox"
              className="Checkbox-toggle"
              name="isRedactorEnabled"
              checked={this.state.redactorEnabled}
              onChange={(e) => {
                this.handleEnableRedactor(e);
              }}
            />
          </div>
        </div>
      </div>
    );
  }
}

export default RedactorRow;
