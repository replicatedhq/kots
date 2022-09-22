import * as React from "react";
import Helmet from "react-helmet";
import { withRouter, Link } from "react-router-dom";

import Toggle from "../shared/Toggle";
import Loader from "../shared/Loader";
import SupportBundleRow from "./SupportBundleRow";
import GenerateSupportBundle from "./GenerateSupportBundle";
import ConfigureRedactorsModal from "./ConfigureRedactorsModal";
import ErrorModal from "../modals/ErrorModal";
import { Utilities } from "../../utilities/utilities";

import "../../scss/components/troubleshoot/SupportBundleList.scss";
import Icon from "../Icon";

class SupportBundleList extends React.Component {
  state = {
    supportBundles: [],
    loading: false,
    errorMsg: "",
    displayRedactorModal: false,
    displayErrorModal: false,
  };

  componentDidMount() {
    this.listSupportBundles();
  }

  listSupportBundles = () => {
    this.setState({ loading: true, errorMsg: "", displayErrorModal: false });

    fetch(
      `${process.env.API_ENDPOINT}/troubleshoot/app/${this.props.watch?.slug}/supportbundles`,
      {
        headers: {
          Authorization: Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "GET",
      }
    )
      .then(async (res) => {
        if (!res.ok) {
          this.setState({
            loading: false,
            errorMsg: `Unexpected status code: ${res.status}`,
            displayErrorModal: true,
          });
          return;
        }
        const response = await res.json();
        this.setState({
          supportBundles: response.supportBundles,
          loading: false,
          errorMsg: "",
          displayErrorModal: false,
        });
      })
      .catch((err) => {
        console.log(err);
        this.setState({
          loading: false,
          errorMsg: err
            ? err.message
            : "Something went wrong, please try again.",
          displayErrorModal: true,
        });
      });
  };

  toggleErrorModal = () => {
    this.setState({ displayErrorModal: !this.state.displayErrorModal });
  };

  toggleRedactorModal = () => {
    this.setState({
      displayRedactorModal: !this.state.displayRedactorModal,
    });
  };

  render() {
    const { watch } = this.props;
    const { loading, errorMsg, supportBundles } = this.state;

    const appTitle = watch.watchName || watch.name;
    const downstream = watch.downstream;

    if (loading) {
      return (
        <div className="flex1 flex-column justifyContent--center alignItems--center">
          <Loader size="60" />
        </div>
      );
    }

    let bundlesNode;
    if (downstream) {
      if (supportBundles?.length) {
        bundlesNode = supportBundles
          .sort((a, b) => new Date(b.createdAt) - new Date(a.createdAt))
          .map((bundle) => (
            <SupportBundleRow
              key={bundle.id}
              bundle={bundle}
              watchSlug={watch.slug}
              isAirgap={watch.isAirgap}
              isSupportBundleUploadSupported={
                watch.isSupportBundleUploadSupported
              }
              refetchBundleList={this.listSupportBundles}
            />
          ));
      } else {
        return <GenerateSupportBundle watch={watch} />;
      }
    }

    return (
      <div className="centered-container u-paddingBottom--30 u-paddingTop--30 flex1 flex">
        <Helmet>
          <title>{`${appTitle} Troubleshoot`}</title>
        </Helmet>
        <div className="flex1 flex-column">
          <div className="flex justifyContent--center u-paddingBottom--30">
            <Toggle
              items={[
                {
                  title: "Support bundles",
                  onClick: () =>
                    this.props.history.push(
                      `/app/${this.props.watch.slug}/troubleshoot`
                    ),
                  isActive: true,
                },
                {
                  title: "Redactors",
                  onClick: () =>
                    this.props.history.push(
                      `/app/${this.props.watch.slug}/troubleshoot/redactors`
                    ),
                  isActive: false,
                },
              ]}
            />
          </div>
          <div className="flex flex1">
            <div className="flex1 flex-column">
              <div className="u-position--relative flex-auto u-paddingBottom--10 flex">
                <div className="flex flex1">
                  <div className="flex1 u-flexTabletReflow">
                    <div className="flex flex1">
                      <div className="flex-auto alignSelf--center">
                        <h2 className="u-fontSize--larger u-fontWeight--bold u-textColor--primary flex alignContent--center">
                          Support bundles
                        </h2>
                      </div>
                    </div>
                    <div className="RightNode flex-auto flex alignItems--center u-position--relative">
                      <Link
                        to={`${this.props.match.url}/generate`}
                        className="btn secondary"
                      >
                        Generate a support bundle
                      </Link>
                      <span
                        className="replicated-link flex alignItems--center u-fontSize--small u-marginLeft--20"
                        onClick={this.toggleRedactorModal}
                      >
                        <Icon
                          icon="marker-tip-outline"
                          size={18}
                          className="clickable u-marginRight--5"
                        />
                        Configure redaction
                      </span>
                    </div>
                  </div>
                </div>
              </div>
              <div
                className={`${
                  watch.downstream ? "flex1 flex-column u-overflow--auto" : ""
                }`}
              >
                {bundlesNode}
              </div>
            </div>
          </div>
        </div>
        {this.state.displayRedactorModal && (
          <ConfigureRedactorsModal onClose={this.toggleRedactorModal} />
        )}
        {errorMsg && (
          <ErrorModal
            errorModal={this.state.displayErrorModal}
            toggleErrorModal={this.toggleErrorModal}
            errMsg={errorMsg}
            tryAgain={this.listSupportBundles}
            err="Failed to get bundles"
            loading={this.state.loading}
            appSlug={this.props.match.params.slug}
          />
        )}
      </div>
    );
  }
}

export default withRouter(SupportBundleList);
