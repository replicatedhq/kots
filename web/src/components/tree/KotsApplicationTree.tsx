import { Component } from "react";
import { withRouter } from "@src/utilities/react-router-utilities";
import { Utilities } from "../../utilities/utilities";
import { KotsPageTitle } from "@components/Head";
import isEmpty from "lodash/isEmpty";
import keys from "lodash/keys";
import MonacoEditor from "@monaco-editor/react";
import Modal from "react-modal";
import CodeSnippet from "../shared/CodeSnippet";

import FileTree from "../shared/FileTree";

import "../../scss/components/troubleshoot/FileTree.scss";

// Types
import { App, KotsParams } from "@types";
import { useLocation, useNavigate } from "react-router-dom";

type Props = {
  params: KotsParams;
  location: ReturnType<typeof useLocation>;
  navigate: ReturnType<typeof useNavigate>;
  outletContext: {
    app: App;
    appName: string;
    appNameSpace: string;
  };
  isEmbeddedCluster: boolean;
};

type State = {
  files: {
    [key: string]: string;
  };
  selectedFile: string;
  displayInstructionsModal: boolean;
  applicationTree: object[];
};
class KotsApplicationTree extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = {
      files: {},
      selectedFile: "/",
      displayInstructionsModal: false,
      applicationTree: [],
    };
  }

  fetchApplicationTree = () => {
    const url = `${process.env.API_ENDPOINT}/app/${this.props.params.slug}/sequence/${this.props.params.sequence}/contents`;
    fetch(url, {
      credentials: "include",
      method: "GET",
    })
      .then((res) => res.json())
      .then(async (res) => {
        const files = res?.files || {};
        const paths = keys(files);
        const applicationTree = Utilities.arrangeIntoApplicationTree(paths);
        if (this.props.location.search) {
          this.setState({
            selectedFile: `/skippedFiles/${this.props.location.search.slice(
              1
            )}`,
          });
        }
        this.setState({
          files,
          applicationTree,
        });
      })
      .catch((err) => {
        throw err;
      });
  };

  componentDidUpdate(lastProps: Props) {
    if (
      this.props.params.slug != lastProps.params.slug ||
      this.props.params.sequence != lastProps.params.sequence
    ) {
      this.fetchApplicationTree();
    }
  }

  componentDidMount() {
    this.fetchApplicationTree();
  }

  toggleInstructionsModal = () => {
    this.setState({
      displayInstructionsModal: !this.state.displayInstructionsModal,
    });
  };

  setSelectedFile = (path: string) => {
    this.setState({
      selectedFile: path,
    });
  };

  back = () => {
    this.props.navigate(-1);
  };

  render() {
    const { displayInstructionsModal, files, applicationTree, selectedFile } =
      this.state;

    const contents = files[selectedFile]
      ? new Buffer(files[selectedFile], "base64").toString()
      : "";

    return (
      <div className="flex-column flex1 ApplicationTree--wrapper u-paddingBottom--30" data-testid="view-files-page">
        <KotsPageTitle pageName="View Files" showAppSlug />
        {!this.props.isEmbeddedCluster && (
          <div className="edit-files-banner u-fontSize--small u-fontWeight--medium">
            Need to edit these files?{" "}
            <span
              onClick={this.toggleInstructionsModal}
              className="u-fontWeight--bold u-cursor--pointer u-textDecoration--underlineOnHover"
            >
              Click here
            </span>{" "}
            to learn how
          </div>
        )}
        <div className="flex flex1 u-marginLeft--30 u-marginRight--30 u-marginTop--10">
          <div className="flex1 dirtree-wrapper flex-column u-overflow-hidden">
            <div className="u-overflow--auto dirtree" data-testid="file-tree">
              <FileTree
                files={applicationTree}
                isRoot={true}
                handleFileSelect={this.setSelectedFile}
                selectedFile={this.state.selectedFile}
              />
              {isEmpty(applicationTree) && (
                <ul className="FileTree-wrapper">
                  <li>Loading file explorer</li>
                </ul>
              )}
            </div>
          </div>
          <div className="AceEditor flex1 flex-column file-contents-wrapper u-position--relative" data-testid="file-editor">
            {this.state.selectedFile === "" ||
            this.state.selectedFile === "/" ? (
              <div className="flex-column flex1 alignItems--center justifyContent--center">
                <p className="u-textColor--bodyCopy u-fontSize--normal u-fontWeight--medium" data-testid="file-editor-empty-state">
                  Select a file from the file explorer to view it here.
                </p>
              </div>
            ) : (
              <MonacoEditor
                language={"yaml"}
                value={contents}
                options={{
                  readOnly: true,
                  contextmenu: false,
                  minimap: {
                    enabled: false,
                  },
                  scrollBeyondLastLine: false,
                }}
              />
            )}
          </div>
        </div>
        {displayInstructionsModal && (
          <Modal
            isOpen={displayInstructionsModal}
            onRequestClose={this.toggleInstructionsModal}
            shouldReturnFocusAfterClose={false}
            contentLabel="Display edit instructions modal"
            ariaHideApp={false}
            className="DisplayInstructionsModal--wrapper Modal MediumSize"
          >
            <div className="Modal-body">
              <h2 className="u-fontSize--largest u-textColor--primary u-fontWeight--bold u-lineHeight--normal">
                Edit patches for your kots application
              </h2>
              <div className="flex flex1 u-marginTop--20">
                <div className="flex-auto">
                  <span className="instruction-modal-number">1</span>
                </div>
                <div className="flex1">
                  <p className="u-fontSize--large u-fontWeight--bold u-textColor--primary u-marginBottom--5 u-lineHeight--normal">
                    Download your application bundle.
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
                    {`kubectl kots download --namespace ${this.props.outletContext.appNameSpace} --slug ${this.props.params.slug}`}
                  </CodeSnippet>
                </div>
              </div>

              <div className="flex flex1 u-marginTop--30">
                <div className="flex-auto">
                  <span className="instruction-modal-number">2</span>
                </div>
                <div className="flex1">
                  <p className="u-fontSize--large u-fontWeight--bold u-textColor--primary u-marginBottom--5 u-lineHeight--normal">
                    Edit any of your files in your favorite IDE.
                  </p>
                </div>
              </div>

              <div className="flex flex1 u-marginTop--30">
                <div className="flex-auto">
                  <span className="instruction-modal-number">3</span>
                </div>
                <div className="flex1">
                  <p className="u-fontSize--large u-fontWeight--bold u-textColor--primary u-marginBottom--5 u-lineHeight--normal">
                    Upload your edited application bundle.
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
                    {`kubectl kots upload --namespace ${this.props.outletContext.appNameSpace} --slug ${this.props.params.slug} ./${this.props.params.slug}`}
                  </CodeSnippet>
                </div>
              </div>
              <div className="u-marginTop--30 flex">
                <button
                  onClick={this.toggleInstructionsModal}
                  className="btn blue primary"
                >
                  Ok, got it!
                </button>
              </div>
            </div>
          </Modal>
        )}
      </div>
    );
  }
}

/* eslint-disable */
// @ts-ignore
export default withRouter(KotsApplicationTree) as any;
/* eslint-enable */
