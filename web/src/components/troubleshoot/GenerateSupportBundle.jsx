import * as React from "react";
import { KotsPageTitle } from "@components/Head";
import { Link } from "react-router-dom";
import { withRouter } from "@src/utilities/react-router-utilities";
import Modal from "react-modal";

import Toggle from "../shared/Toggle";
import SupportBundleCollectProgress from "../troubleshoot/SupportBundleCollectProgress";
import CodeSnippet from "@src/components/shared/CodeSnippet";
import UploadSupportBundleModal from "../troubleshoot/UploadSupportBundleModal";
import ConfigureRedactorsModal from "./ConfigureRedactorsModal";
import ErrorModal from "../modals/ErrorModal";
import { Utilities } from "../../utilities/utilities";
import { Repeater } from "../../utilities/repeater";

import "../../scss/components/troubleshoot/GenerateSupportBundle.scss";
import Icon from "../Icon";

class GenerateSupportBundle extends React.Component {
  constructor(props) {
    super(props);

    this.state = {
      displayUploadModal: false,
      totalBundles: null,
      showRunCommand: false,
      isGeneratingBundle: false,
      displayRedactorModal: false,
      loadingSupportBundles: false,
      supportBundles: [],
      listSupportBundlesJob: new Repeater(),
      newBundleSlug: "",
      bundleAnalysisProgress: {},
      errorMsg: "",
      displayErrorModal: false,
      networkErr: false,
    };
  }

  componentDidMount() {
    this.checkIfSupportBundleIsBeingGenerated();
  }

  componentWillUnmount() {
    this.state.listSupportBundlesJob.stop();
  }

  componentDidUpdate(lastProps, lastState) {
    const { watch, navigate, bundle } = this.props;
    const { totalBundles, loadingSupportBundles, supportBundles, networkErr } =
      this.state;

    if (!loadingSupportBundles) {
      if (totalBundles === null) {
        this.setState({
          totalBundles: supportBundles?.length,
        });
        this.state.listSupportBundlesJob.start(this.listSupportBundles, 2000);
        return;
      }

      // this is needed to redirect to the support bundle page
      // after collecting a support bundle from the CLI is done
      if (this.state.listSupportBundlesJob.isRunning()) {
        if (supportBundles?.length > totalBundles) {
          const bundle = supportBundles[0]; // safe. there's at least 1 element in this array.
          if (bundle.status !== "running") {
            this.state.listSupportBundlesJob.stop();
            if (bundle.status === "failed") {
              navigate(`/app/${watch.slug}/troubleshoot`);
            } else {
              navigate(`/app/${watch.slug}/troubleshoot/analyze/${bundle.id}`);
            }
          }
        }
      }
    }

    if (networkErr !== lastState.networkErr) {
      if (networkErr) {
        this.state.listSupportBundlesJob.stop();
      } else {
        this.state.listSupportBundlesJob.start(this.listSupportBundles, 2000);
        return;
      }
    }
  }

  checkIfSupportBundleIsBeingGenerated() {
    this.setState({
      loadingSupportBundles: true,
      errorMsg: "",
      displayErrorModal: false,
      networkErr: false,
    });

    fetch(
      `${process.env.API_ENDPOINT}/troubleshoot/app/${this.props.watch?.slug}/supportbundles`,
      {
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        method: "GET",
      }
    )
      .then(async (res) => {
        if (!res.ok) {
          this.setState({
            loadingSupportBundles: false,
            errorMsg: `Unexpected status code: ${res.status}`,
            displayErrorModal: true,
            networkErr: false,
          });
          return;
        }
        const response = await res.json();
        let bundleRunning = false;
        if (response.supportBundles) {
          bundleRunning = response.supportBundles.find(
            (bundle) => bundle.status === "running"
          );
        }
        if (bundleRunning) {
          this.props.updateBundleSlug(bundleRunning.slug);
          this.setState({
            newBundleSlug: bundleRunning.slug,
            isGeneratingBundle: true,
            generateBundleErrMsg: "",
            supportBundles: response.supportBundles,
            loadingSupportBundles: false,
            errorMsg: "",
            displayErrorModal: false,
            networkErr: false,
          });
        } else {
          this.setState({
            supportBundles: response.supportBundles,
            loadingSupportBundles: false,
            errorMsg: "",
            displayErrorModal: false,
            networkErr: false,
          });
        }
      })
      .catch((err) => {
        console.log(err);
        this.setState({
          loadingSupportBundles: false,
          errorMsg: err
            ? err.message
            : "Something went wrong, please try again.",
          displayErrorModal: true,
          networkErr: true,
        });
      });
  }

  listSupportBundles = () => {
    return new Promise((resolve, reject) => {
      this.setState({
        loadingSupportBundles: true,
        errorMsg: "",
        displayErrorModal: false,
        networkErr: false,
      });

      console.log(this.props.watch?.slug, " slug");
      fetch(
        `${process.env.API_ENDPOINT}/troubleshoot/app/${this.props.watch?.slug}/supportbundles`,
        {
          headers: {
            "Content-Type": "application/json",
          },
          credentials: "include",
          method: "GET",
        }
      )
        .then(async (res) => {
          if (!res.ok) {
            this.setState({
              loadingSupportBundles: false,
              errorMsg: `Unexpected status code: ${res.status}`,
              displayErrorModal: true,
              networkErr: false,
            });
            return;
          }
          const response = await res.json();
          this.setState({
            supportBundles: response.supportBundles,
            loadingSupportBundles: false,
            errorMsg: "",
            displayErrorModal: false,
            networkErr: false,
          });

          resolve();
        })
        .catch((err) => {
          console.log(err);
          this.setState({
            loadingSupportBundles: false,
            errorMsg: err
              ? err.message
              : "Something went wrong, please try again.",
            displayErrorModal: true,
            networkErr: true,
          });
          reject(err);
        });
    });
  };

  showCopyToast(message, didCopy) {
    this.setState({
      showToast: didCopy,
      copySuccess: didCopy,
      copyMessage: message,
    });
    setTimeout(() => {
      this.setState({
        showToast: false,
        copySuccess: false,
        copyMessage: "",
      });
    }, 3000);
  }

  renderIcons = (shipOpsRef, gitOpsRef) => {
    if (shipOpsRef) {
      return <Icon icon="kots-o-filled" size={18} />;
    } else if (gitOpsRef) {
      return <Icon icon="github-icon" size={19} />;
    } else {
      return;
    }
  };

  toggleShow = (ev, section) => {
    ev.preventDefault();
    this.setState({
      [section]: !this.state[section],
    });
  };

  getLabel = ({ shipOpsRef, gitOpsRef, title }) => {
    return (
      <div style={{ alignItems: "center", display: "flex" }}>
        <span style={{ fontSize: 18, marginRight: "0.5em" }}>
          {this.renderIcons(shipOpsRef, gitOpsRef)}
        </span>
        <span style={{ fontSize: 14 }}>{title}</span>
      </div>
    );
  };

  collectBundle = (clusterId) => {
    const { watch, navigate } = this.props;

    let url = `${process.env.API_ENDPOINT}/troubleshoot/supportbundle/app/${watch?.id}/cluster/${clusterId}/collect`;
    if (!watch.id) {
      // TODO: check if helm managed, not if id is missing
      url = `${process.env.API_ENDPOINT}/troubleshoot/supportbundle/app/${watch?.slug}/collect`;
    }

    fetch(url, {
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
      method: "POST",
    })
      .then(async (res) => {
        if (!res.ok) {
          this.setState({
            isGeneratingBundle: false,
            generateBundleErrMsg: `Unable to generate bundle: Status ${res.status}`,
          });
          return;
        }
        const response = await res.json();
        this.props.updateBundleSlug(response.slug);
        this.setState({ newBundleSlug: response.slug });

        navigate(`/app/${watch.slug}/troubleshoot/analyze/${response.slug}`);
        this.setState({
          isGeneratingBundle: true,
          generateBundleErrMsg: "",
        });
      })
      .catch((err) => {
        console.log(err);
        this.setState({
          isGeneratingBundle: false,
          generateBundleErrMsg: err
            ? err.message
            : "Something went wrong, please try again.",
        });
      });
  };

  fetchSupportBundleCommand = async () => {
    const { watch } = this.props;

    const res = await fetch(
      `${process.env.API_ENDPOINT}/troubleshoot/app/${watch.slug}/supportbundlecommand`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          origin: window.location.origin,
        }),
      }
    );
    if (!res.ok) {
      throw new Error(`Unexpected status code: ${res.status}`);
    }
    const response = await res.json();
    this.setState({
      showRunCommand: !this.state.showRunCommand,
      bundleCommand: response.command,
    });
  };

  toggleErrorModal = () => {
    this.setState({ displayErrorModal: !this.state.displayErrorModal });
  };

  toggleModal = () => {
    this.setState({
      displayUploadModal: !this.state.displayUploadModal,
    });
  };

  toggleRedactorModal = () => {
    this.setState({
      displayRedactorModal: !this.state.displayRedactorModal,
    });
  };

  render() {
    const {
      displayUploadModal,
      showRunCommand,
      isGeneratingBundle,
      generateBundleErrMsg,
      errorMsg,
    } = this.state;
    const { watch, navigate } = this.props;
    const appTitle = watch.downstream?.currentVersion?.appTitle || watch.name;

    return (
      <div className="GenerateSupportBundle--wrapper container flex-column u-overflow--auto u-paddingTop--30 u-paddingBottom--20 alignItems--center">
        <KotsPageTitle pageName="Generate Support Bundle" showAppSlug />
        <div className="GenerateSupportBundle">
          {!watch.downstream && !this.state.supportBundles?.length ? (
            <Link
              to={`/watch/${watch.slug}/troubleshoot`}
              className="link u-marginRight--5"
            >
              {" "}
              &lt; Support Bundle List{" "}
            </Link>
          ) : null}
          <div className="u-marginTop--15">
            <div className="flex justifyContent--center u-paddingBottom--30">
              <Toggle
                items={[
                  {
                    title: "Support bundles",
                    onClick: () =>
                      navigate(`/app/${this.props.watch.slug}/troubleshoot`),
                    isActive: true,
                  },
                  {
                    title: "Redactors",
                    onClick: () =>
                      navigate(
                        `/app/${this.props.watch.slug}/troubleshoot/redactors`
                      ),
                    isActive: false,
                  },
                ]}
              />
            </div>
            <div className="card-bg u-padding--15">
              <div className="flex justifyContent--spaceBetween u-paddingBottom--15">
                <p className="card-title">Support Bundles</p>
                <span
                  className="link flex alignItems--center u-fontSize--small u-marginLeft--20"
                  onClick={this.toggleRedactorModal}
                >
                  <Icon
                    icon="marker-tip-outline"
                    size={18}
                    className="clickable u-marginRight--5"
                  />
                  Configure redactors
                </span>
              </div>
              <div className="card-item GenerateSupportBundleDetails u-padding--50">
                <h2 className="u-fontSize--jumbo2 u-fontWeight--bold u-textColor--primary u-textAlign--center u-paddingBottom--15 break-word">
                  Analyze {appTitle} for support
                </h2>
                <p className="u-fontSize--normal u-textColor--bodyCopy u-lineHeight--medium u-marginTop--5 u-textAlign--center u-fontWeight--medium break-word">
                  To diagnose any problems with the application, click the
                  button below to get started. This will collect logs, resources
                  and other data from the running application and analyze them
                  against a set of known problems in {appTitle}. Logs, cluster
                  info and other data will not leave your cluster.
                </p>
                {!isGeneratingBundle && (
                  <div className="flex alignItems--center justifyContent--center u-marginTop--30">
                    <button
                      className="btn primary blue break-word"
                      type="button"
                      onClick={this.collectBundle.bind(
                        this,
                        watch.downstream?.cluster?.id
                      )}
                    >
                      Analyze {appTitle}
                    </button>
                  </div>
                )}
              </div>
              <div className="flex1 flex-column u-margin--auto">
                {showRunCommand ? (
                  <div>
                    <div className="u-marginTop--15">
                      <h2 className="u-fontSize--larger u-fontWeight--bold u-textColor--primary">
                        Run this command in your cluster
                      </h2>
                      <CodeSnippet
                        language="bash"
                        canCopy={true}
                        onCopyText={
                          <span className="u-textColor--success">
                            Command has been copied to your clipboard
                          </span>
                        }
                      >
                        {this.state.bundleCommand}
                      </CodeSnippet>
                    </div>
                    <div className="u-marginTop--15">
                      <button
                        className="btn secondary"
                        type="button"
                        onClick={this.toggleModal}
                      >
                        {" "}
                        Upload a support bundle{" "}
                      </button>
                    </div>
                  </div>
                ) : (
                  <div className="u-marginTop--15 u-fontSize--normal">
                    If you'd prefer,{" "}
                    <a
                      href="#"
                      onClick={(e) => this.fetchSupportBundleCommand()}
                    >
                      click here
                    </a>{" "}
                    to get a command to manually generate a support bundle.
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
        <Modal
          isOpen={displayUploadModal}
          onRequestClose={this.toggleModal}
          shouldReturnFocusAfterClose={false}
          ariaHideApp={false}
          contentLabel="GenerateBundle-Modal"
          className="Modal MediumSize"
        >
          <div className="Modal-body">
            <UploadSupportBundleModal
              watch={this.props.watch}
              onBundleUploaded={(bundleId) => {
                const url = `/app/${this.props.params.slug}/troubleshoot/analyze/${bundleId}`;
                navigate(url);
              }}
            />
          </div>
        </Modal>
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
            loading={this.state.loadingSupportBundles}
            appSlug={this.props.params.slug}
          />
        )}
      </div>
    );
  }
}

export default withRouter(GenerateSupportBundle);
