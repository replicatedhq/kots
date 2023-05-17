import React, { useEffect, useReducer } from "react";
import Modal from "react-modal";
import CodeSnippet from "@components/shared/CodeSnippet";
import { useNavigate } from "react-router-dom";
import { App, LicenseFile, KotsParams, SupportBundle } from "@types";
// @ts-ignore
import Dropzone from "react-dropzone";
import Icon from "@components/Icon";
import isEmpty from "lodash/isEmpty";
// @ts-ignore
import randomstring from "randomstring";
import { useRouteMatch } from "react-router";
import { Repeater } from "../../utilities/repeater";

type Props = {
  isOpen: boolean;
  toggleModal: () => void;
  watch: App | null;
  updateBundleSlug: (value: string) => void;
};

type State = {
  bundleCommand: string;
  fileUploading: boolean;
  generateBundleErrMsg: string;
  showGetBundleSpec: boolean;
  supportBundleFile: LicenseFile | null;
  uploadBundleErrMsg: string;
  supportBundles: SupportBundle[] | null;
  listSupportBundlesJob: Repeater;
  errorMsg: string;
  totalBundles: number | null;
  loadingSupportBundles: boolean;
};

const GenerateSupportBundleModal = ({
  isOpen,
  toggleModal,
  watch,
  updateBundleSlug,
}: Props) => {
  const [state, setState] = useReducer(
    (currentState: State, newState: Partial<State>) => ({
      ...currentState,
      ...newState,
    }),
    {
      bundleCommand: "",
      fileUploading: false,
      generateBundleErrMsg: "",
      showGetBundleSpec: false,
      supportBundleFile: {} as LicenseFile,
      uploadBundleErrMsg: "",
      supportBundles: null,
      listSupportBundlesJob: new Repeater(),
      errorMsg: "",
      totalBundles: null,
      loadingSupportBundles: false,
    }
  );

  const toggleShowGetBundleSpec = () => {
    setState({ showGetBundleSpec: !state.showGetBundleSpec });
  };

  const navigate = useNavigate();
  const match = useRouteMatch<KotsParams>();

  const fetchSupportBundleCommand = async () => {
    const res = await fetch(
      `${process.env.API_ENDPOINT}/troubleshoot/app/${watch?.slug}/supportbundlecommand`,
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
    setState({ bundleCommand: response.command });
  };

  const listSupportBundles = () => {
    setState({ loadingSupportBundles: true });
    return new Promise<void>((resolve, reject) => {
      fetch(
        `${process.env.API_ENDPOINT}/troubleshoot/app/${watch?.slug}/supportbundles`,
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
              errorMsg: `Unable to list support bundles: Status ${res.status}`,
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
            setState({
              loadingSupportBundles: false,
              errorMsg: "",
            });
          } else {
            setState({
              supportBundles: response.supportBundles,
              loadingSupportBundles: false,
              errorMsg: "",
            });
          }
          resolve();
        })
        .catch((err) => {
          setState({
            errorMsg: err
              ? err.message
              : "Something went wrong, please try again.",
          });
          reject(err);
        });
    });
  };

  useEffect(() => {
    fetchSupportBundleCommand();
    listSupportBundles();

    return () => {
      state.listSupportBundlesJob.stop();
    };
  }, []);

  useEffect(() => {
    const { totalBundles, supportBundles, listSupportBundlesJob } = state;
    if (supportBundles && supportBundles.length > 0) {
      if (totalBundles === null) {
        setState({ totalBundles: supportBundles?.length });
        listSupportBundlesJob.start(listSupportBundles, 2000);
        return;
      } else if (listSupportBundlesJob.isRunning()) {
        if (supportBundles?.length > totalBundles) {
          const bundle = supportBundles[0];
          if (bundle.status !== "running") {
            listSupportBundlesJob.stop();
            if (bundle.status === "failed") {
              navigate(`/app/${watch?.slug}/troubleshoot`);
            } else {
              navigate(`/app/${watch?.slug}/troubleshoot/analyze/${bundle.id}`);
            }
          }
        }
      }
    }
  }, [state.supportBundles]);

  const collectBundle = (clusterId: number | undefined) => {
    let url = `${process.env.API_ENDPOINT}/troubleshoot/supportbundle/app/${watch?.id}/cluster/${clusterId}/collect`;
    if (!watch?.id) {
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
          setState({
            generateBundleErrMsg: `Unable to generate bundle: Status ${res.status}`,
          });
        }
        const response = await res.json();
        updateBundleSlug(response.slug);

        navigate(`/app/${watch?.slug}/troubleshoot/analyze/${response.slug}`);
        setState({ generateBundleErrMsg: "" });
      })
      .catch((err) => {
        console.log(err);

        setState({
          generateBundleErrMsg: err
            ? err.message
            : "Something went wrong, please try again.",
        });
      });
  };

  const onDrop = (files: LicenseFile[]) => {
    setState({ supportBundleFile: files[0] });
  };

  const uploadAndAnalyze = async () => {
    try {
      const bundleId = randomstring.generate({ capitalization: "lowercase" });
      const uploadBundleUrl = `${process.env.API_ENDPOINT}/troubleshoot/${watch?.id}/${bundleId}`;

      setState({ fileUploading: true, uploadBundleErrMsg: "" });

      const response = await fetch(uploadBundleUrl, {
        method: "PUT",
        // using JSON.stringify(supportBundle) here will cause the request to fail
        // @ts-ignore
        body: state.supportBundle,
        headers: {
          "Content-Type": "application/tar+gzip",
        },
      });

      if (!response.ok) {
        setState({
          fileUploading: false,
          uploadBundleErrMsg: `Unable to upload the bundle: Status ${response.status}`,
        });

        return;
      }

      setState({ fileUploading: false, uploadBundleErrMsg: "" });
      toggleModal();
      const url = `/app/${match.params.slug}/troubleshoot/analyze/${bundleId}`;
      navigate(url);
    } catch (err) {
      setState({
        fileUploading: false,
        uploadBundleErrMsg: err
          ? `Unable to upload the bundle: ${(err as Error)?.message}`
          : "Something went wrong, please try again.",
      });
    }
  };

  const hasFile = state.supportBundleFile && !isEmpty(state.supportBundleFile);
  const clearFile = () => {
    setState({ supportBundleFile: null });
  };

  return (
    <Modal
      isOpen={isOpen}
      className="Modal generate-support-modal"
      shouldReturnFocusAfterClose={false}
      contentLabel="Connection terminated modal"
      onRequestClose={toggleModal}
      ariaHideApp={false}
    >
      <div className="u-padding--25" onClick={(e) => e.stopPropagation()}>
        <span className="u-fontWeight--medium card-title u-fontSize--larger">
          Generate a support bundle
        </span>
        <div className="analyze-modal">
          <span className="u-fontWeight--bold u-textColor--primary">
            Analyze {watch?.name}
          </span>
          <div className="flex analyze-content alignItems--center justifyContent--spaceBetween">
            <p
              style={{ maxWidth: "450px" }}
              className="u-fontSize--normal u-lineHeight--normal"
            >
              Collect logs, resources and other data from the application and
              analyze them against a set of known criteria to check. Data will
              not leave your cluster.
            </p>
            <div>
              <button
                type="button"
                className="btn primary"
                onClick={() => collectBundle(watch?.downstream?.cluster?.id)}
              >
                Analyze
              </button>
              {state.generateBundleErrMsg && (
                <p className="u-textColor--error u-marginTop--10 u-fontSize--normal">
                  {state.generateBundleErrMsg}
                </p>
              )}
            </div>
          </div>
        </div>
        {state.showGetBundleSpec ? (
          <div className="u-marginTop--15 command-modal">
            <span className="u-fontWeight--bold u-textColor--primary">
              Run a command to generate a support bundle
            </span>
            {state.errorMsg && (
              <p className="u-textColor--error u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-marginBottom--10">
                {state.errorMsg}
              </p>
            )}
            <div className="u-marginTop--15">
              {state.showGetBundleSpec && (
                <>
                  <p className="u-paddingBottom--10 u-textColor--secondary u-fontSize--normal">
                    Run the following commands to generate a support bundle from
                    the CLI. You can then upload a support bundle so that it
                    appears in the admin console.
                  </p>
                  <CodeSnippet
                    language="bash"
                    canCopy={true}
                    onCopyText={
                      <span className="u-textColor--success">
                        Command has been copied to your clipboard
                      </span>
                    }
                  >
                    {state.bundleCommand}
                  </CodeSnippet>
                </>
              )}
            </div>
            <div
              className="flex u-marginTop--30 justifyContent--center alignItems--center"
              style={{ gap: "10px" }}
            >
              <div
                style={{ borderBottom: "1px solid #BEBEBE", width: "180px" }}
              ></div>
              <span>or</span>
              <div
                style={{ borderBottom: "1px solid #BEBEBE", width: "180px" }}
              ></div>
            </div>
          </div>
        ) : (
          <div className="u-marginTop--15">
            <span className="u-fontSize--normal">
              Or{" "}
              <a href="#" onClick={toggleShowGetBundleSpec}>
                click here
              </a>{" "}
              to get a command to manually generate a support bundle. This is
              useful if the admin console is inaccessible.
            </span>
          </div>
        )}

        <div
          className={`u-marginTop--30 FileUpload-wrapper ${
            hasFile ? "has-file" : ""
          }`}
        >
          {state.uploadBundleErrMsg && (
            <p className="u-textColor--error u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-marginBottom--10">
              {state.uploadBundleErrMsg}
            </p>
          )}
          <Dropzone
            className="Dropzone-wrapper"
            accept="application/gzip, .gz"
            onDropAccepted={onDrop}
            multiple={false}
          >
            {!hasFile && (
              <div className="u-textAlign--center">
                <Icon
                  icon="yaml-icon"
                  size={40}
                  className="u-marginBottom--10 gray-color"
                />
                <p className="u-fontSize--normal u-textColor--secondary u-fontWeight--medium u-lineHeight--normal">
                  Drag your bundle here or{" "}
                  <span className="u-linkColor u-fontWeight--medium u-textDecoration--underlineOnHover">
                    choose a file to upload
                  </span>
                </p>
              </div>
            )}
          </Dropzone>
          {hasFile && (
            <div
              className="flex flexDirection--column justifyContent--spaceBetween"
              style={{ gap: "15px" }}
            >
              <div className={`${hasFile ? "has-file-border" : ""}`}>
                <p className="u-fontSize--normal u-fontWeight--medium ">
                  {state.supportBundleFile?.name}
                </p>
              </div>

              <div className="flex flex-column justifyContent--center">
                <button
                  type="button"
                  className="btn secondary blue nowrap"
                  onClick={uploadAndAnalyze}
                  disabled={state.fileUploading || !hasFile}
                >
                  {state.fileUploading ? "Uploading" : "Upload support bundle"}
                </button>
                <span
                  className="replicated-link u-fontSize--small u-marginTop--10 u-textAlign--center"
                  onClick={clearFile}
                >
                  Select a different file
                </span>
              </div>
            </div>
          )}
        </div>

        <div className="flex u-marginTop--30">
          <button className="btn primary" onClick={toggleModal}>
            Ok, got it!
          </button>
        </div>
      </div>
    </Modal>
  );
};

export default GenerateSupportBundleModal;
