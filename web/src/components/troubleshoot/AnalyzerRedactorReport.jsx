import React, { Component } from "react";
import { withRouter } from "react-router-dom";
import isEmpty from "lodash/isEmpty";

import AnalyzerRedactorReportRow from "./AnalyzerRedactorReportRow";
import Loader from "../shared/Loader";
import ErrorModal from "../modals/ErrorModal";
import { Utilities } from "../../utilities/utilities";


export class AnalyzerRedactorReport extends Component {
  constructor(props) {
    super(props);
    this.state = {
      redactions: {},
      displayErrorModal: false
    };
  }

  getSupportBundleRedactions = () => {
    this.setState({
      isLoadingRedactions: true,
      redactionsErrMsg: "",
      displayErrorModal: false
    });

    fetch(`${window.env.API_ENDPOINT}/troubleshoot/supportbundle/${this.props.bundle?.id}/redactions`, {
      method: "GET",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      }
    })
      .then(res => res.json())
      .then(result => {
        if (result.success) {
          this.setState({
            redactions: result.redactions,
            isLoadingRedactions: false,
            redactionsErrMsg: "",
            displayErrorModal: false
          })
        } else {
          this.setState({
            isLoadingRedactions: false,
            redactionsErrMsg: result.error,
            displayErrorModal: true
          })
        }
      })
      .catch(err => {
        this.setState({
          isLoadingRedactions: false,
          redactionsErrMsg: err.message ? err.message : "There was an error while showing the redactor report. Please try again",
          displayErrorModal: true
        })
      })
  }

  toggleErrorModal = () => {
    this.setState({ displayErrorModal: !this.state.displayErrorModal });
  }

  componentDidMount() {
    if (this.props.bundle) {
      this.getSupportBundleRedactions();
    }
  }

  render() {
    const { isLoadingRedactions, redactionsErrMsg, redactions } = this.state;

    if (isLoadingRedactions) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      )
    }

    return (
      <div className="flex flex-column">
        <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-lineHeight--normal u-marginTop--10 u-marginBottom--20">Below is a list of default redactors that were applied when collecting this support bundle. You can see how many files each redactor affected and how many values were redacted.</p>
        {!isEmpty(redactions) && Object.keys(redactions?.byRedactor).map((redactor) => (
          <AnalyzerRedactorReportRow
            key={`redactor-${redactor}`}
            redactor={redactor}
            match={this.props.match}
            history={this.props.history}
            redactorFiles={redactions?.byRedactor[redactor]}
          />
        ))}
        {redactionsErrMsg &&
          <ErrorModal
            errorModal={this.state.displayErrorModal}
            toggleErrorModal={this.toggleErrorModal}
            errMsg={redactionsErrMsg}
            tryAgain={this.getSupportBundleRedactions}
            err="Failed to get redactors"
            loading={this.state.isLoadingRedactions}
            appSlug={this.props.match.params.slug}
          />}
      </div>
    );
  }
}

export default withRouter(AnalyzerRedactorReport);
