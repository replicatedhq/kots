import * as React from "react";
import PropTypes from "prop-types";

const CardEmptyState = ({ toggleModal, watchName }) => (
  <div className="EmptyState--wrapper justifyContent--center alignItems--center flex-column flex1">
    <div className="flex flex1 completed-steps-wrapper u-marginBottom--5">
      <div className="flex-column flex1 completed-step alignItems--center u-textAlign--center">
        <p className="command u-color--dustyGray"><span className="icon checkmark-icon u-marginRight--5"></span>Ship init</p>
        <span className="icon image-wrapper init"></span>
        <p className="u-color--dustyGray">{watchName} has been configured for your application</p>
      </div>
      <div className="flex-column flex1 completed-step alignItems--center u-textAlign--center">
        <p className="command u-color--dustyGray"><span className="icon checkmark-icon u-marginRight--5"></span>Ship watch</p>
        <span className="icon image-wrapper watch"></span>
        <p className="u-color--dustyGray">We're observing upstream {watchName} for updates</p>
      </div>
      <div className="flex-column flex1 completed-step alignItems--center u-textAlign--center">
        <p className="command u-color--tundora">Ship update</p>
        <span className="icon image-wrapper update"></span>
        <p className="u-color--dustyGray">Configure GitHub to make automatic PRs</p>
      </div>
    </div>
    <div className="flex-auto flex justifyContent--center u-marginTop--10">
      <button onClick={toggleModal} className="btn primary">Add a deployment</button>
    </div>
  </div>
);

CardEmptyState.propTypes = {
  toggleModal: PropTypes.func.isRequired
}

export default CardEmptyState;
