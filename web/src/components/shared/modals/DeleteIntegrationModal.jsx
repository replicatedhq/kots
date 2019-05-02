import * as React from "react";
import { withRouter } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import Modal from "react-modal";
import { deleteNotification } from "../../../mutations/WatchMutations";


class DeleteIntegrationModal extends React.Component {
  constructor() {
    super();
    this.state = {
      deleting: false
    };
  }
	
  handleDeleteIntegration = (id, isPending) => {
    this.setState({ deleting: true });
    this.props.deleteNotification(id, isPending)
      .then(() => { 
        this.setState({ deleting: false });
        this.props.toggleDeleteIntegrationModal("","");
        if(this.props.submitCallback && typeof this.props.submitCallback === "function") {
          this.props.submitCallback();
        }
      })
      .catch(() => { 
        this.setState({ deleting: false });
      });
  }

  render() {
    const {
      displayDeleteIntegrationModal,
      toggleDeleteIntegrationModal,
      integrationToDeleteId,
      integrationToDeletePath,
      isPending
    } = this.props;
		
    return (
      <Modal
        isOpen={displayDeleteIntegrationModal}
        onRequestClose={toggleDeleteIntegrationModal}
        shouldReturnFocusAfterClose={false}
        ariaHideApp={false}
        contentLabel="Modal"
        className="Modal DefaultSize"
      >
        <div className="Modal-body flex flex-column flex1 u-overflow--auto">
          <h2 className="u-fontSize--largest u-color--tuna u-fontWeight--bold u-lineHeight--normal">Delete GitHub integration</h2>
          <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">Are you sure you want to delete this GitHub integration?
            <span className="u-color--tundora u-fontWeight--bold u-fontSize--normal alignItems--self"> {integrationToDeletePath} </span>
          </p>
          <div className="Form">
            <div className="flex flex1 u-marginBottom--30">
              <div className="flex flex1 flex-column u-marginRight--10">
                <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal">You'll no longer be able to manage this GitHub integration, and it will be removed from this view.</p>
              </div>
            </div>
            <div className="flex justifyContent--flexEnd u-marginTop--20">
              <button className="btn secondary u-marginRight--10" onClick={toggleDeleteIntegrationModal}>Cancel</button>
              <button className="btn primary" disabled={this.state.deleting} onClick={() => { this.handleDeleteIntegration(integrationToDeleteId, isPending)}}>
                {this.state.deleting ? "Deleting..." : "Delete"}
              </button>
            </div>
          </div>
        </div>
      </Modal>
    );
  }
}
export default withRouter(compose(
  withApollo,
  graphql(deleteNotification, {
    props: ({ mutate }) => ({
      deleteNotification: (id, isPending) => mutate({ variables: { id, isPending } })
    })
  })
)(DeleteIntegrationModal));