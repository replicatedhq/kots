import { Component } from "react";
import AceEditor from "react-ace";
import Select from "react-select";
import Loader from "../shared/Loader";
import yaml from "js-yaml";
import "brace/ext/searchbox";

export class PodAnalyzerDetails extends Component {
  state = {
    activeTab: "podLogs",
    podContainers: [],
    podEvents: "",
    podDefinition: "",
    selectedContainer: {},
    selectedContainerLogs: "",
    loading: false,
    errMsg: "",
  };

  togglePodDetailView = (active) => {
    this.setState({ activeTab: active });
  };

  componentDidMount() {
    this.getPodDetails();
  }

  componentDidUpdate(_, lastState) {
    if (this.state.activeTab !== lastState.activeTab && this.aceEditor) {
      this.aceEditor.editor.resize(true);
    }
  }

  getPodDetails = async () => {
    const { pod } = this.props;
    this.setState({ loading: true, errMsg: "" });

    fetch(
      `${process.env.API_ENDPOINT}/troubleshoot/app/qakots/supportbundle/${this.props.bundleId}/pod?podNamespace=${pod.namespace}&podName=${pod.name}`,
      {
        method: "GET",
        credentials: "include",
      }
    )
      .then(async (result) => {
        const data = await result.json();

        let selectedContainer = {};
        let groupedPodOptions = [];
        const initContainers = data.podContainers.filter(
          (c) => c.isInitContainer
        );
        const regContainers = data.podContainers.filter(
          (c) => !c.isInitContainer
        );
        if (initContainers.length > 0) {
          groupedPodOptions.push({
            label: "Init containers",
            options: initContainers,
          });
        }
        if (regContainers.length > 0) {
          groupedPodOptions.push({
            label: "Containers",
            options: regContainers,
          });
        }
        selectedContainer = groupedPodOptions[0].options[0];
        this.onSelectedContainerChange(selectedContainer);
        this.setState({
          loading: false,
          podContainers: groupedPodOptions,
          podDefinition: yaml.dump(data.podDefinition),
          podEvents: yaml.dump(data.podEvents),
        });
      })
      .catch((err) => {
        this.setState({
          loading: false,
          errMsg: err,
        });
      });
  };

  onSelectedContainerChange = (selectedContainer) => {
    this.setState({ selectedContainer: selectedContainer });

    this.setState({
      loading: true,
      errMsg: "",
    });

    fetch(
      `${process.env.API_ENDPOINT}/troubleshoot/supportbundle/${
        this.props.bundleId
      }/files?filename=${encodeURIComponent(selectedContainer.logsFilePath)}`,
      {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
      }
    )
      .then(async (result) => {
        const data = await result.json();
        const decodedLogs = Buffer.from(
          data.files[selectedContainer.logsFilePath],
          "base64"
        ).toString();
        this.setState({ loading: false, selectedContainerLogs: decodedLogs });
      })
      .catch((err) => {
        this.setState({
          loading: false,
          errMsg: err,
        });
      });
  };

  renderPodDetailView = () => {
    switch (this.state.activeTab) {
      case "podDefinition":
        return (
          <div className="flex1 u-border--gray">
            {this.renderEditor(
              "definition",
              this.state.podDefinition,
              "yaml",
              "Definition not found"
            )}
          </div>
        );
      case "podLogs":
        return (
          <div>
            {this.state.podContainers && this.state.podContainers.length > 1 ? (
              <div className="flex flex1 alignItems--center u-marginBottom--15">
                <p className="u-fontSize--normal u-fontWeight--medium u-textColor--header u-lineHeight--normal u-marginRight--10">
                  Which container logs would you like to view?
                </p>
                <div className="flex-auto">
                  <Select
                    className="replicated-select-container app"
                    classNamePrefix="replicated-select"
                    options={this.state.podContainers}
                    getOptionLabel={(container) => container.name}
                    getOptionValue={(container) => container.logsFilePath}
                    value={this.state.selectedContainer}
                    onChange={this.onSelectedContainerChange}
                    isOptionSelected={(container) => {
                      container.name === this.state.selectedContainer.name;
                    }}
                  />
                </div>
              </div>
            ) : (
              <p className="u-fontSize--normal u-fontWeight--medium u-textColor--header u-lineHeight--normal u-marginBottom--10">
                Viewing logs for the "
                {this.state.podContainers[0]?.options[0].name}"{" "}
                {this.state.podContainers.length > 0 &&
                this.state.podContainers[0]?.options[0]?.isInitContainer
                  ? "init container)"
                  : "container"}
              </p>
            )}
            <div className="flex1 u-border--gray">
              {this.renderEditor(
                "logs",
                this.state.selectedContainerLogs,
                "text",
                "No logs found"
              )}
            </div>
          </div>
        );
      case "podEvents":
        return (
          <div className="flex1 u-border--gray">
            {this.renderEditor(
              "events",
              this.state.podEvents,
              "yaml",
              "No events found"
            )}
          </div>
        );
      default:
        return <div>nothing selected</div>;
    }
  };

  renderEditor = (key, content, mode, emptyMsg) => {
    const editorHeight = 500;

    if (this.state.loading) {
      return (
        <div
          style={{ height: editorHeight }}
          className="flex-column flex1 alignItems--center justifyContent--center"
        >
          <Loader size="60" />
        </div>
      );
    }

    if (!content) {
      return (
        <div
          style={{ height: editorHeight }}
          className="flex-column flex1 alignItems--center justifyContent--center"
        >
          <p className="u-fontSize--large u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--more">
            {emptyMsg}
          </p>
        </div>
      );
    }

    return (
      <AceEditor
        ref={(el) => (this.aceEditor = el)}
        key={key}
        mode={mode}
        theme="chrome"
        className="flex1 flex"
        readOnly={true}
        value={content}
        height={`${editorHeight}px`}
        width="100%"
        editorProps={{
          $blockScrolling: Infinity,
          useSoftTabs: true,
          tabSize: 2,
        }}
        setOptions={{
          scrollPastEnd: false,
          showGutter: true,
        }}
      />
    );
  };

  render() {
    const { pod } = this.props;
    return (
      <div className="flex1 flex-column">
        <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--more">
          Details for Pod: {pod.namespace}/{pod.name}
        </p>
        <div className="SupportBundleTabs--wrapper flex-column flex1">
          <div className="flex tab-items">
            <span
              className={`${
                this.state.activeTab === "podLogs" ? "is-active" : ""
              } tab-item blue`}
              onClick={() => this.togglePodDetailView("podLogs")}
            >
              Pod logs
            </span>
            <span
              className={`${
                this.state.activeTab === "podEvents" ? "is-active" : ""
              } tab-item blue`}
              onClick={() => this.togglePodDetailView("podEvents")}
            >
              Pod events
            </span>
            <span
              className={`${
                this.state.activeTab === "podDefinition" ? "is-active" : ""
              } tab-item blue`}
              onClick={() => this.togglePodDetailView("podDefinition")}
            >
              Pod definition
            </span>
          </div>
          <div className="flex flex1 action-content">
            <div className="flex1 flex-column file-contents-wrapper u-position--relative">
              {this.renderPodDetailView()}
            </div>
          </div>
        </div>
      </div>
    );
  }
}

export default PodAnalyzerDetails;
