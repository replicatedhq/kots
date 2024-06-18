import { useReducer, useEffect, useContext } from "react";
import { KotsPageTitle } from "@components/Head";
import { withRouter } from "@src/utilities/react-router-utilities";

import Toggle from "../shared/Toggle";
import Loader from "../shared/Loader";
import SupportBundleRow from "./SupportBundleRow";
import GenerateSupportBundle from "./GenerateSupportBundle";
import ConfigureRedactorsModal from "./ConfigureRedactorsModal";
import ErrorModal from "../modals/ErrorModal";
import { Repeater } from "@src/utilities/repeater";
import "../../scss/components/troubleshoot/SupportBundleList.scss";
import Icon from "../Icon";
import { App, SupportBundle, SupportBundleProgress } from "@types";
import GenerateSupportBundleModal from "./GenerateSupportBundleModal";
import { useNavigate, useOutletContext, useParams } from "react-router-dom";
import { ToastContext } from "@src/context/ToastContext";
import Toast from "@components/shared/Toast";
import { usePrevious } from "@src/hooks/usePrevious";
import { useSelectedApp } from "@features/App";

type Props = {
  bundle: SupportBundle;
  bundleProgress: SupportBundleProgress;
  displayErrorModal: boolean;
  loading: boolean;
  loadingBundle: boolean;
  loadingBundleId: string;
  pollForBundleAnalysisProgress: () => void;
  updateBundleSlug: (slug: string) => void;
  updateState: (value: Object) => void;
  watch: App | null;
  isEmbeddedClusterEnabled: boolean;
};

type State = {
  bundleAnalysisProgress?: SupportBundleProgress;
  displayRedactorModal: boolean;
  errorMsg?: string;
  loadingBundleId?: string;
  loadingSupportBundles: boolean;
  pollForBundleAnalysisProgress: Repeater;
  supportBundles?: SupportBundle[];
  isGeneratingBundleOpen: boolean;
};

export const SupportBundleList = (props: Props) => {
  const [state, setState] = useReducer(
    (currentState: State, newState: Partial<State>) => ({
      ...currentState,
      ...newState,
    }),
    {
      displayRedactorModal: false,
      loadingSupportBundles: false,
      pollForBundleAnalysisProgress: new Repeater(),
      isGeneratingBundleOpen: false,
    }
  );

  const navigate = useNavigate();
  const {
    deleteBundleId,
    isToastVisible,
    toastMessage,
    toastType,
    setIsToastVisible,
    toastChild,
  } = useContext(ToastContext);

  // rework this so full page refresh is not needed.
  // const deleteBundleFromList = (deleteId: string) => {
  //   setState({
  //     supportBundles: state.supportBundles?.filter(
  //       (bundle) => bundle.id !== deleteId
  //     ),
  //   });
  // };

  const outletContext: Props = useOutletContext();

  const params = useParams();

  const listSupportBundles = () => {
    setState({
      errorMsg: "",
    });

    outletContext.updateState({
      loading: true,
      displayErrorModal: true,
      loadingBundle: false,
    });

    fetch(
      `${process.env.API_ENDPOINT}/troubleshoot/app/${params?.slug}/supportbundles`,
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
          setState({
            errorMsg: `Unexpected status code: ${res.status}`,
          });
          outletContext.updateState({
            loading: false,
            displayErrorModal: true,
          });
          return;
        }
        const response = await res.json();
        let bundleRunning = false;
        if (response.supportBundles) {
          bundleRunning = response.supportBundles.find(
            (bundle: SupportBundle) => bundle.status === "running"
          );
        }
        if (bundleRunning) {
          state.pollForBundleAnalysisProgress.start(
            outletContext.pollForBundleAnalysisProgress,
            1000
          );
        }
        setState({
          supportBundles: response.supportBundles,
          errorMsg: "",
        });
        outletContext.updateState({
          loading: false,
          displayErrorModal: false,
        });
      })
      .catch((err) => {
        console.log(err);
        setState({
          errorMsg: err
            ? err.message
            : "Something went wrong, please try again.",
        });
        outletContext.updateState({
          displayErrorModal: true,
          loading: false,
        });
      });
  };

  useEffect(() => {
    listSupportBundles();
    return () => {
      state.pollForBundleAnalysisProgress.stop();
    };
  }, []);

  useEffect(() => {
    if (state.supportBundles && state.supportBundles.length < 1) {
      navigate(`/app/${outletContext.watch?.slug}/troubleshoot/generate`);
    }
  }, [state.supportBundles]);

  useEffect(() => {
    const { bundle } = outletContext;
    if (bundle?.status !== "running") {
      listSupportBundles();
      state.pollForBundleAnalysisProgress.stop();
      if (bundle.status === "failed") {
        navigate(`/app/${outletContext.watch?.slug}/troubleshoot`);
      }
    }
  }, [outletContext.bundle]);

  const prevLoadingBundleId = usePrevious(outletContext.loadingBundleId);
  const prevDeleteBundleId = usePrevious(deleteBundleId);

  useEffect(() => {
    // if the current bundle to delete is the same as the bundle that is loading
    // stop the polling
    if (outletContext.loadingBundleId === deleteBundleId) {
      state.pollForBundleAnalysisProgress.stop();
      outletContext.updateState({
        loadingBundleId: "",
        loadingBundle: false,
      });
    }
    // if the loading bundle is done and user previously tried to delete the bundle, and changed their mind (undo)
    // refresh the list
    if (
      prevLoadingBundleId === "" &&
      prevDeleteBundleId !== "" &&
      deleteBundleId === ""
    ) {
      listSupportBundles();
    }
    // if the loading bundle is not done and user tried to delete a bundle, and changed their mind (undo)
    // refresh the list, which will start polling again, and show the progress bar
    if (prevLoadingBundleId === prevDeleteBundleId && deleteBundleId === "") {
      outletContext.updateState({
        loadingBundleId: prevLoadingBundleId,
        loadingBundle: true,
      });
      listSupportBundles();
      // need to refresh show the progress bar
    }
  }, [deleteBundleId]);

  const toggleGenerateBundleModal = () => {
    setState({
      isGeneratingBundleOpen: !state.isGeneratingBundleOpen,
    });
  };

  const toggleErrorModal = () => {
    outletContext.updateState({
      displayErrorModal: !outletContext.displayErrorModal,
    });
  };

  const toggleRedactorModal = () => {
    setState({
      displayRedactorModal: !state.displayRedactorModal,
    });
  };

  const { watch, loading, loadingBundle } = outletContext;
  const { errorMsg, supportBundles, isGeneratingBundleOpen } = state;
  const selectedApp = useSelectedApp();
  const downstream = selectedApp?.downstream;

  if (loading) {
    return (
      <div className="flex1 flex-column justifyContent--center alignItems--center">
        <Loader size="60" />
      </div>
    );
  }

  let bundlesNode;
  if (downstream && watch) {
    if (supportBundles?.length) {
      bundlesNode = supportBundles
        .sort(
          (a, b) =>
            new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
        )
        .map((bundle) => (
          <SupportBundleRow
            key={bundle.id}
            bundle={bundle}
            watchSlug={selectedApp.slug}
            isAirgap={selectedApp?.isAirgap}
            isSupportBundleUploadSupported={
              watch?.isSupportBundleUploadSupported
            }
            refetchBundleList={listSupportBundles}
            //  deleteBundleFromList={deleteBundleFromList}
            progressData={
              outletContext?.loadingBundleId === bundle.id
                ? outletContext.bundleProgress
                : null
            }
            loadingBundle={
              outletContext.loadingBundleId === bundle.id &&
              outletContext.loadingBundle
            }
            className={bundle.id === deleteBundleId ? "deleting" : ""}
            isCustomer={false}
          />
        ));
    } else {
      return (
        <GenerateSupportBundle
          watch={selectedApp}
          updateBundleSlug={outletContext.updateBundleSlug}
          bundle={props.bundle}
          pollForBundleAnalysisProgress={
            outletContext.pollForBundleAnalysisProgress
          }
          isEmbeddedClusterEnabled={props.isEmbeddedClusterEnabled}
        />
      );
    }
  }

  return (
    <>
      <div className="centered-container u-paddingBottom--30 u-paddingTop--30 flex1 flex">
        <KotsPageTitle pageName="Version History" showAppSlug />
        <div className="flex1 flex-column">
          <div className="flex justifyContent--center u-paddingBottom--30">
            <Toggle
              items={[
                {
                  title: "Support bundles",
                  onClick: () =>
                    navigate(`/app/${outletContext?.watch?.slug}/troubleshoot`),
                  isActive: true,
                },
                {
                  title: "Redactors",
                  onClick: () =>
                    navigate(
                      `/app/${outletContext?.watch?.slug}/troubleshoot/redactors`
                    ),
                  isActive: false,
                },
              ]}
            />
          </div>
          <div className="card-bg support-bundle-list-wrapper">
            <div className="flex flex1 flex-column">
              <div className="u-position--relative flex-auto u-paddingBottom--10 flex">
                <div className="flex flex1 u-flexTabletReflow">
                  <div className="flex flex1">
                    <div className="flex-auto alignSelf--center">
                      <p className="card-title">Support bundles</p>
                    </div>
                  </div>
                  <div className="RightNode flex-auto flex alignItems--center u-position--relative">
                    <a
                      onClick={() =>
                        !loadingBundle && toggleGenerateBundleModal()
                      }
                      className={`replicated-link flex alignItems--center u-fontSize--small ${
                        loadingBundle ? "generating-bundle" : ""
                      }`}
                    >
                      <Icon
                        icon="tools"
                        size={18}
                        className="clickable u-marginRight--5"
                      />
                      Generate a support bundle
                    </a>
                    <span
                      className="link flex alignItems--center u-fontSize--small u-marginLeft--20"
                      onClick={toggleRedactorModal}
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
              <div
                className={`${
                  watch?.downstream ? "flex1 flex-column u-overflow--auto" : ""
                }`}
              >
                {bundlesNode}
              </div>
            </div>
          </div>
        </div>
        {state.displayRedactorModal && (
          <ConfigureRedactorsModal onClose={toggleRedactorModal} />
        )}
        {errorMsg && (
          <ErrorModal
            errorModal={outletContext.displayErrorModal}
            toggleErrorModal={toggleErrorModal}
            errMsg={errorMsg}
            tryAgain={listSupportBundles}
            err="Failed to get bundles"
            loading={outletContext.loading}
            appSlug={params.slug}
          />
        )}
        <GenerateSupportBundleModal
          isOpen={isGeneratingBundleOpen}
          toggleModal={toggleGenerateBundleModal}
          selectedApp={selectedApp}
          updateBundleSlug={outletContext.updateBundleSlug}
          isEmbeddedClusterEnabled={props.isEmbeddedClusterEnabled}
        />
      </div>

      <Toast isToastVisible={isToastVisible} type={toastType}>
        <div className="tw-flex tw-items-center">
          <p className="tw-ml-2 tw-mr-4">{toastMessage}</p>
          {toastChild}
          <Icon
            icon="close"
            size={10}
            className="tw-mx-4 tw-cursor-pointer"
            onClick={() => setIsToastVisible(false)}
          />
        </div>
      </Toast>
    </>
  );
};

/* eslint-disable */
// @ts-ignore
export default withRouter(SupportBundleList) as any;
/* eslint-enable */
