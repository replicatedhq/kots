import React, { Component } from "react";
import { Link, withRouter } from "react-router-dom";
import isEmpty from "lodash/isEmpty";

import AnalyzerRedactorReportRow from "./AnalyzerRedactorReportRow";
import Loader from "../shared/Loader";
import { Utilities } from "../../utilities/utilities";


export class AnalyzerRedactorReport extends Component {
  constructor(props) {
    super(props);
    this.state = {
      redactions: {}
    };
  }

  gtSupportBundleRedactions = () => {
    this.setState({
      isLoadingRedations: true,
      redactionsErrMsg: "",
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
            isLoadingRedations: false,
            redactionsErrMsg: "",
          })
        } else {
          this.setState({
            isLoadingRedations: false,
            redactionsErrMsg: result.error,
          })
        }
      })
      .catch(err => {
        console.log(err);
        this.setState({
          isLoadingRedations: false,
          redactionsErrMsg: err.message ? err.message : "There was an error while showing the redactor report. Please try again",
        })
      })
  }

  componentDidMount() {
    if (this.props.bundle) {
      this.gtSupportBundleRedactions();
    }
  }

  render() {
    const { isLoadingRedations, redactionsErrMsg, redactions } = this.state;

    if (isLoadingRedations) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      )
    }

    if (redactionsErrMsg) {
      return (
        <div class="flex1 flex-column justifyContent--center alignItems--center">
          <span className="icon redWarningIcon" />
          <p className="u-color--chestnut u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-marginTop--10">{redactionsErrMsg}</p>
        </div>
      )
    }

    return (
      <div className="flex flex-column">
        <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-lineHeight--normal u-marginTop--small u-marginBottom--20">Below is a list of default redactors that were applied when collecting this support bundle. You can see how many files each redactor affected and how many values were redacted.</p>
        {!isEmpty(redactions) && Object.keys(redactions?.byRedactor).map((redactor) => ( 
          <AnalyzerRedactorReportRow
            key={`redactor-${redactor}`}
            redactor={redactor}
            match={this.props.match}
            history={this.props.history}
            redactorFiles={redactions?.byRedactor[redactor]}
          />  
        ))}
      </div>
    );
  }
}

export default withRouter(AnalyzerRedactorReport);
