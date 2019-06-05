import React from "react";
import classNames from "classnames";
import ReactTooltip from "react-tooltip"
import { withRouter } from "react-router-dom";
import { compose, withApollo } from "react-apollo";
import PropTypes from "prop-types";
import WatchContributorsModal from "../shared/modals/WatchContributorsModal";
import "../../scss/components/watches/WatchContributors.scss"

class WatchContributors extends React.Component {
  constructor() {
    super();
    this.state = {
      displayContributorsModal: false,
      editContributorsFor: {}
    }
  }

  static propTypes = {
    contributors: PropTypes.array.isRequired,
    watchId: PropTypes.string.isRequired,
    watchName: PropTypes.string.isRequired,
    maxVisible: PropTypes.number,
    title: PropTypes.string
  }

  toggleContributorsModal = (watch) => {
    this.setState({
      displayContributorsModal: !this.state.displayContributorsModal,
      editContributorsFor: this.state.displayContributorsModal ? { id: "", name: "" } : watch
    });
  }

  render() {
    const {
      contributors,
      watchId,
      watchName,
      title,
      maxVisible,
      className
    } = this.props;
    const max = maxVisible || 3;
    let _contributors = [];
    
    if (contributors) {
      _contributors = contributors.length > max ? contributors.slice(0,max) : contributors;
    }
    
    const remainingContributors = contributors && contributors.length > max ? (contributors.length - max) : 0
    return (
      <div className={classNames("WatchContributors--wrapper flex-column", className)}>
        {title && (
          <div className="flex justifyContent--spaceBetween">
            <p className="uppercase-title">{title}</p>
            <span
              className="u-fontSize--small replicated-link" 
              onClick={() => this.toggleContributorsModal({ id: watchId, name: watchName })}>
              Manage
            </span>
          </div>
          
        ) }
        <div className="flex-column">
          <p className="u-fontSize--jumbo1 u-fontWeight--bold u-color--tuna u-marginTop--15">Individuals</p>
          <div className="flex u-marginTop--15">
            {contributors && contributors.length && _contributors.map((contributor, i) => (
              <div key={i} className="watch-icon-wrapper u-cursor--pointer" onClick={() => this.toggleContributorsModal({ id: watchId, name: watchName })} data-tip={`${contributor.login}-${i}`} data-for={`${contributor.login}-${i}`}>
                <span className="contributer-icon" style={{ backgroundImage: `url(${contributor.avatar_url})` }}></span>
                <ReactTooltip id={`${contributor.login}-${i}`} effect="solid" className="replicated-tooltip">
                  <span>{contributor.login}</span>
                </ReactTooltip>
              </div>
            ))}
          </div>
          {contributors && contributors.length > max ?
            <div className="flex-column justifyContent--center u-cursor--pointer u-marginLeft--5 u-marginRight--5" onClick={() => this.toggleContributorsModal({ id: watchId, name: watchName })}>
              <span className="u-color--tundora u-fontSize--small u-fontWeight--medium">+ {remainingContributors} other{remainingContributors === 1 ? "" : "s"}</span>
            </div>
            : null}
          <div className="flex-column flex1 justifyContent--center u-marginLeft--5">
          </div>
        </div>
        {this.state.displayContributorsModal &&
          <WatchContributorsModal
            displayContributorsModal={this.state.displayContributorsModal}
            toggleContributorsModal={() => { this.toggleContributorsModal() }}
            watchBeingEdited={this.state.editContributorsFor}
            submitCallback={() => {
              if (typeof this.props.watchCallback === "function") {
                this.props.watchCallback();
              }
            }}
          />
        }
      </div>
    );
  }
}

export default compose(
  withApollo,
  withRouter
)(WatchContributors);
