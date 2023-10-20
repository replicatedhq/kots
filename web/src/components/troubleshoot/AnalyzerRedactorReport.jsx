import { Component } from "react";
import { withRouter } from "@src/utilities/react-router-utilities";
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
      displayErrorModal: false,
    };
  }

  getSupportBundleRedactions = () => {
    this.setState({
      isLoadingRedactions: true,
      redactionsErrMsg: "",
      displayErrorModal: false,
    });

    fetch(
      `${process.env.API_ENDPOINT}/troubleshoot/supportbundle/${this.props.outletContext.bundle?.id}/redactions`,
      {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
      }
    )
      .then((res) => res.json())
      .then((result) => {
        if (result.success) {
          this.setState({
            redactions: result.redactions,
            isLoadingRedactions: false,
            redactionsErrMsg: "",
            displayErrorModal: false,
          });
        } else {
          this.setState({
            isLoadingRedactions: false,
            redactionsErrMsg: result.error,
            displayErrorModal: true,
          });
        }
      })
      .catch((err) => {
        this.setState({
          isLoadingRedactions: false,
          redactionsErrMsg: err.message
            ? err.message
            : "There was an error while showing the redactor report. Please try again",
          displayErrorModal: true,
        });
      });
  };

  toggleErrorModal = () => {
    this.setState({ displayErrorModal: !this.state.displayErrorModal });
  };

  componentDidMount() {
    if (this.props.outletContext.bundle) {
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
      );
    }

    return (
      <div className="flex flex-column">
        {!isEmpty(redactions) ? (
          <p className="u-fontSize--normal u-textColor--bodyCopy u-fontWeight--medium u-lineHeight--normal u-marginTop--10 u-marginBottom--20">
            Below is a list of the redactor specs that were applied when
            collecting this support bundle. You can see how many files each spec
            affected and how many values were redacted.
          </p>
        ) : (
          <p className="u-fontSize--normal u-textColor--bodyCopy u-fontWeight--medium u-lineHeight--normal u-marginTop--10 u-marginBottom--20">
            This support bundle does not contain information about redactors
            that were applied during collection.
          </p>
        )}
        {!isEmpty(redactions) &&
          Object.keys(redactions?.byRedactor).map((redactor) => (
            <AnalyzerRedactorReportRow
              key={`redactor-${redactor}`}
              redactor={redactor}
              redactorFiles={redactions?.byRedactor[redactor]}
            />
          ))}
        {redactionsErrMsg && (
          <ErrorModal
            errorModal={this.state.displayErrorModal}
            toggleErrorModal={this.toggleErrorModal}
            errMsg={redactionsErrMsg}
            tryAgain={this.getSupportBundleRedactions}
            err="Failed to get redactors"
            loading={this.state.isLoadingRedactions}
            appSlug={this.props.params.slug}
          />
        )}
      </div>
    );
  }
}

export default withRouter(AnalyzerRedactorReport);
