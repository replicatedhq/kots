import * as React from "react";
import Loader from "./shared/Loader";

export default class ShipLoading extends React.Component {
  render() {
    return (
      <div className="Form flex-column flex1 alignItems--center justifyContent--center">
        <div className="init-pre-wrapper flex-auto">
          <div className="flex1 flex-column u-textAlign--center">
            <p className="u-fontSize--larger u-color--tuna u-fontWeight--bold u-lineHeight--normal u-wordBreak--all">{this.props.headerText}</p>
            {this.props.subText && <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">{this.props.subText}</p>}
            <Loader size="50" />
          </div>
        </div>
      </div>
    );
  }
}
