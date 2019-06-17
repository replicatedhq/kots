import * as React from "react";
import PropTypes from "prop-types";
import { Link } from "react-router-dom";
import isEmpty from "lodash/isEmpty";

const CardHeader = ({ watch, watchIntegrations, onEditApplication, isPending }) => (
  <div className="installed-watch-header flex">
    <div className="installed-watch-name flex alignItems--center">
      <div className="logo" style={{ backgroundImage: `url(${watch.watchIcon})` }}></div>
      <h2 className="u-fontSize--jumbo u-fontWeight--bold u-color--tuna">{ watch.watchName }</h2>
    </div>
    {!isEmpty(watch.currentVersion) &&
      <div className="installed-watch-integrations flex flex-column u-marginRight--30 u-paddingRight--10">
        <p className="uppercase-title">Current version</p>
        <div className="flex alignItems--center">
          <p className="integration-number flex u-fontSize--large u-color--tuna u-fontWeight--bold">
            { watch.currentVersion.title }
          </p>
          <Link to={`/watch/${watch.slug}/history`} className="replicated-link u-marginLeft--5 u-fontSize--small">Version history</Link>
        </div>
      </div>
    }
    {watchIntegrations && (watchIntegrations.email.length || watchIntegrations.webhook.length) ?
      <div className="installed-watch-integrations flex flex-column">
        <p className="uppercase-title">Integrations</p>
        <div className="flex alignItems--center">
          <p className="integration-number flex u-fontSize--large u-color--tuna u-fontWeight--bold">
            <span className="icon flex alignItems--center integration-card-icon-email"></span>
            { watchIntegrations.email.length }
          </p>
          <p className="border">|</p>
          <p className="flex alignItems--center u-fontSize--large u-color--tuna u-fontWeight--bold">
            <span className="icon integration-card-icon-webhook"></span>
            { watchIntegrations.webhook.length }
          </p>
          <Link to={`/watch/${watch.slug}/integrations`} className="u-marginLeft--10 u-fontSize--small replicated-link">Manage</Link>
        </div>
      </div>
    : null}
    <div className="installed-watch-actions alignItems--center flex">
      {!isPending && <Link to={`/watch/${watch.slug}`} className="u-marginRight--10 u-fontSize--small replicated-link">Application Details</Link>}
      <button className="btn secondary green" onClick={() => onEditApplication(watch)}>{isPending ? "Install application" : "Edit application"}</button>
    </div>
  </div>
);

CardHeader.propTypes = {
  watch: PropTypes.object.isRequired,
  watchIntegrations: PropTypes.object.isRequired,
  onEditApplication: PropTypes.func.isRequired
}

export default CardHeader;
