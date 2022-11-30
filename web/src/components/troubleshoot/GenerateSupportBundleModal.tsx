import React, { useEffect } from "react";
import Modal from "react-modal";
import CodeSnippet from "@components/shared/CodeSnippet";
import { Utilities } from "@src/utilities/utilities";
import { useHistory } from "react-router";
import { App } from "@types";

type Props = {
  isOpen: boolean;
  toggleModal: () => void;
  watch: App | null;
  updateBundleSlug: (value: string) => void;
};

const GenerateSupportBundleModal = ({
  isOpen,
  toggleModal,
  watch,
  updateBundleSlug,
}: Props) => {
  const [showGetBundleSpec, setShowGetBundleSpec] = React.useState(false);
  const [bundleCommand, setBundleCommand] = React.useState("");
  const toggleShowGetBundleSpec = () => {
    setShowGetBundleSpec(!showGetBundleSpec);
  };
  const [generateBundleErrMsg, setGenerateBundleErrMsg] = React.useState("");
  const history = useHistory();

  const fetchSupportBundleCommand = async () => {
    const res = await fetch(
      `${process.env.API_ENDPOINT}/troubleshoot/app/${watch?.slug}/supportbundlecommand`,
      {
        method: "POST",
        headers: {
          Authorization: Utilities.getToken(),
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          origin: window.location.origin,
        }),
      }
    );
    if (!res.ok) {
      throw new Error(`Unexpected status code: ${res.status}`);
    }
    const response = await res.json();
    setBundleCommand(response.command);
  };

  useEffect(() => {
    fetchSupportBundleCommand();
  }, []);

  const collectBundle = (clusterId: number | undefined) => {
    let url = `${process.env.API_ENDPOINT}/troubleshoot/supportbundle/app/${watch?.id}/cluster/${clusterId}/collect`;
    if (!watch?.id) {
      // TODO: check if helm managed, not if id is missing
      url = `${process.env.API_ENDPOINT}/troubleshoot/supportbundle/app/${watch?.slug}/collect`;
    }

    fetch(url, {
      headers: {
        Authorization: Utilities.getToken(),
        "Content-Type": "application/json",
      },
      method: "POST",
    })
      .then(async (res) => {
        if (!res.ok) {
          setGenerateBundleErrMsg(
            `Unable to generate bundle: Status ${res.status}`
          );
        }
        const response = await res.json();
        updateBundleSlug(response.slug);

        history.push(
          `/app/${watch?.slug}/troubleshoot/analyze/${response.slug}`
        );

        setGenerateBundleErrMsg("");
      })
      .catch((err) => {
        console.log(err);

        setGenerateBundleErrMsg(
          err ? err.message : "Something went wrong, please try again."
        );
      });
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
              Collect logs, resources and other data from the running
              application and analyze them against a set of known problems in
              Sentry Enterprise. Logs, cluster info and other data will not
              leave your cluster.
            </p>
            <div>
              <button
                type="button"
                className="btn primary"
                onClick={() => collectBundle(watch?.downstream?.cluster?.id)}
              >
                Analyze
              </button>
              {generateBundleErrMsg && (
                <p className="u-textColor--error u-marginTop--10 u-fontSize--normal">
                  {generateBundleErrMsg}
                </p>
              )}
            </div>
          </div>
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
        <div className="u-marginTop--30 command-modal">
          <span className="u-fontWeight--bold u-textColor--primary">
            Run a command to generate a support bundle
          </span>
          <p className="u-marginTop--10 u-textColor--secondary u-fontSize--normal u-lineHeight--normal">
            Run the following command passing the correct path to your support
            bundle spec to generate a support bundle. If your vendor did not
            provide a support bundle spec,{" "}
            <a href="#" onClick={toggleShowGetBundleSpec}>
              run this command
            </a>{" "}
            from a machine with internet access.
          </p>
          <div className="u-marginTop--30">
            {showGetBundleSpec && (
              <>
                <p className="u-paddingBottom--10 u-textColor--secondary u-fontSize--normal">
                  Run this command from a machine with internet access to get a
                  support bundle spec
                </p>
                <CodeSnippet
                  language="bash"
                  canCopy
                  onCopyText={
                    <span className="u-textColor--success">
                      Command has been copied to your clipboard
                    </span>
                  }
                >
                  curl -o spec.yaml https://kots.io -H
                  'User-agent:Replicated_Troubleshoot/v1beta1'{" "}
                </CodeSnippet>
              </>
            )}
            <CodeSnippet
              language="bash"
              canCopy={true}
              onCopyText={
                <span className="u-textColor--success">
                  Command has been copied to your clipboard
                </span>
              }
            >
              {bundleCommand}
            </CodeSnippet>
          </div>
          <div className="flex u-marginTop--30">
            <button className="btn secondary blue u-marginRight--15">
              Upload support bundle
            </button>{" "}
            <button className="btn primary" onClick={toggleModal}>
              Ok, got it!
            </button>
          </div>
        </div>
      </div>
    </Modal>
  );
};

export default GenerateSupportBundleModal;
