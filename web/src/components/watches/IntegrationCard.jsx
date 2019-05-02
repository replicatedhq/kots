import * as React from "react";
import { Utilities } from "../../utilities/utilities";

export default class IntegrationCard extends React.Component {

  constructor() {
    super();
    this.state = {
      isEnabled: 0,
      cardDetails: {
        email: {
          title: "Email address",
          description: "Configure ship to send an email when an update is available for a 3rd-party application."
        },
        webhook: {
          title: "Webhook URL",
          description: "Configure ship to ping a webhook when an update is available for a 3rd-party application."
        },
      }
    }
  }

  handleFormChange = (field, e) => {
    let nextState = {};
    const val = e.target.checked ? 1 : 0;
    nextState[field] = val;
    this.setState(nextState);
    this.props.toggleEnable(this.props.item.id, val)
  }

  handleEditClick = (type, id) => {
    this.props.onEditClick(type, id);
  }

  determineTypeToShow = (item) => {
    if (!item) {return;}
    switch (this.props.type) {
    case "webhook":
      return item.webhook.uri;
    default:
      return item.email.recipientAddress;
    }
  }

  emptyCard = () => {
    const { cardDetails } = this.state;
    return (
      <div className="flex-column flex1">
        <div className="flex-column flex1 justifyContent--center alignItems--center">
          <span className={`icon integration-icon-${this.props.type} u-marginTop--10`}></span>
          <p className="u-color--tundora u-fontWeight--bold u-textAlign--center u-fontSize--normal u-marginTop--30">{cardDetails[this.props.type].title}</p>
        </div>
        <div className="u-marginTop--20">
          <p className="u-fontSize--small u-fontWeight--medium u-textAlign--center u-lineHeight--normal u-color--dustyGray">{cardDetails[this.props.type].description}</p>
        </div>
        <div className="button-wrapper flex">
          <div className="flex1 flex card-action-wrapper u-cursor--pointer u-textAlign--center">
            <span className="flex1 card-action u-color--astral u-fontSize--small u-fontWeight--medium" onClick={() => this.handleEditClick(this.props.type, this.props.item.id)}>Configure integration</span>
          </div>
        </div>
      </div>
    )
  }

  componentDidMount() {
    this.setState({ isEnabled: this.props.item.enabled });
  }

  render() {
    const { item, type } = this.props;
    const { cardDetails } = this.state;

    return (
      <div data-qa={`IntegrationCard--${item.id}`} className="integration-card flex-column">
        {!item.updatedOn ? this.emptyCard() :
          <div className="flex-column flex1">
            <div className="flex">
              <span className={`u-marginRight--5 icon integration-card-icon-${type}`}></span>
              <div className="flex-column flex1 justifyContent--center">
                <p className="u-color--tundora u-fontWeight--bold u-fontSize--normal">{cardDetails[type].title}</p>
              </div>
            </div>
            <div className="u-marginTop--20">
              {item.triggeredOn ?
                <p className="u-fontSize--small u-fontWeight--medium u-color--dustyGray">Last triggred on <span className="u-fontWeight--bold">{Utilities.dateFormat(item.triggeredOn, "MMMM D, YYYY")}</span></p>
                :
                <p className="u-fontSize--small u-fontWeight--medium u-color--dustyGray">No events have been sent</p>
              }
            </div>
            <div className="u-marginTop--20">
              <p className="u-fontSize--small u-fontWeight--medium u-color--dustyGray u-marginTop--5">Sending to:</p>
              <p className="u-fontSize--normal u-fontWeight--medium u-color--tundora u-marginTop--5">{this.determineTypeToShow(item)}</p>
            </div>
            <div className="flex u-marginTop--20">
              <div className={`flex flex-auto Checkbox--switch ${this.state.isEnabled === 1 ? "is-checked" : ""}`}>
                <input
                  type="checkbox"
                  className="Checkbox-toggle flex-auto"
                  name={`${type}-enabled`}
                  id={`${type}-enabled`}
                  checked={this.state.isEnabled === 1}
                  onChange={(e) => { this.handleFormChange("isEnabled", e) }} />
              </div>
              <label htmlFor={`${type}-enabled`} className="flex1 flex-column flex-verticalCenter u-marginLeft--5 u-color--tundora u-fontSize--normal u-fontWeight--medium u-cursor--pointer">Enable integration</label>
            </div>
            <div className="button-wrapper flex">
              <div className="flex1 flex card-action-wrapper u-cursor--pointer u-textAlign--center">
                <span className="flex1 card-action u-color--astral u-fontSize--small u-fontWeight--medium" onClick={() => this.handleEditClick(type, item.id)}>Manage integration</span>
              </div>
            </div>
          </div>
        }
      </div>
    );
  }
}
