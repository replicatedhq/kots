import * as React from "react";
import { withRouter } from "react-router-dom";
import AceEditor from "react-ace";
import Select from "react-select";
import Loader from "../shared/Loader";
import yaml from "js-yaml";
import { Utilities } from "../../utilities/utilities";

const POD_LOGS = `2021/10/04 21:42:06 kotsadm version v1.52.0
2021/10/04 21:42:06 Starting monitor loop
Starting Admin Console API on port 3000...
{"level":"error","ts":"2021-10-04T21:42:22Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:42:27Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:42:32Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:42:37Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:42:42Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:42:47Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:42:52Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:42:57Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:43:02Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:43:07Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:43:12Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:43:17Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:43:22Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:43:28Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:43:33Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:43:38Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:43:43Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:43:48Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:43:53Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:43:58Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:44:03Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:44:08Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:44:13Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:44:18Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:44:23Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:44:28Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:44:33Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:44:38Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:44:43Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:44:48Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:44:53Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:44:58Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:45:03Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:45:08Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:45:13Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:45:18Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:45:23Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:45:28Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:45:33Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:45:38Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:45:43Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:45:48Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:45:53Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:45:58Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:46:03Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:46:08Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:46:13Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:46:18Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:46:23Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:46:28Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:46:33Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:46:38Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:46:43Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:46:49Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:46:54Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:46:59Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
`;

export class PodAnalyzerDetails extends React.Component {

  state = {
    activeTab: "podDefinition",
    podContainers: [],
    podEvents: "",
    podDefinition: "",
    selectedContainer: {},
    loading: false,
    errMsg: "",
  }
  
  togglePodDetailView = (active) => {
    this.setState({ activeTab: active });
  }

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
    
    // fetch(`${window.env.API_ENDPOINT}/troubleshoot/app/qakots/supportbundle/${this.props.bundleId}/pod?podNamespace=${pod.involvedObject?.namespace}&podName=${pod.involvedObject?.name}`, {
    fetch(`${window.env.API_ENDPOINT}/troubleshoot/app/qakots/supportbundle/2041y5f3xzi5ewauoaiqogyccme/pod?podNamespace=default&podName=sqs-7449b544fc-mw4dx`, {
      method: "GET",
      headers: {
        "Authorization": Utilities.getToken(),
      }
    })
    .then(async (result) => {
      const data = await result.json();

      const podContainers = [];
      for (const containerName in data.podContainers) {
        podContainers.push({
          name: containerName,
          logsFilePath: data.podContainers[containerName],
        });
      }

      let selectedContainer = {};
      if (podContainers.length > 0) {
        selectedContainer = podContainers[0];
      }

      this.setState({ loading: false, podContainers, selectedContainer, podDefinition: yaml.dump(data.podDefinition), podEvents: yaml.dump(data.podEvents) });
    })
    .catch(err => {
      this.setState({
        loading: false,
        errMsg: err,
      });
    });
  }

  onSelectedContainerChange = (selectedContainer) => {
    this.setState({ selectedContainer: selectedContainer });

    this.setState({
      loading: true,
      errMsg: "",
    });

    fetch(`${window.env.API_ENDPOINT}/troubleshoot/supportbundle/${this.props.bundleId}/files?filename=${encodeURIComponent(selectedContainer.logsFilePath)}`, {
      method: "GET",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      },
    })
    .then(async (result) => {
      const data = await result.json();
      console.log(data)
      this.setState({ loading: false });
    })
    .catch(err => {
      this.setState({
        loading: false,
        errMsg: err,
      });
    });
  }

  renderPodDetailView = () => {
    switch (this.state.activeTab) {
    case "podDefinition":
      return (
        <div className="flex1 u-border--gray">
          {this.renderEditor(this.state.podDefinition, "yaml")}
        </div>
      )
    case "podLogs":
      return (
        <div>
          <p className="u-fontSize--normal u-fontWeight--medium u-textColor--header u-lineHeight--normal u-marginBottom--10">Which container logs would you like to view?</p>
          <div className="u-marginBottom--10 flex-auto">
            <Select
              className="replicated-select-container"
              classNamePrefix="replicated-select"
              options={this.state.podContainers}
              getOptionLabel={(container) => container.name}
              getOptionValue={(container) => container.logsFilePath}
              value={this.state.selectedContainer}
              onChange={this.onSelectedContainerChange}
              isOptionSelected={(container) => { container.name === this.state.selectedContainer.name }}
            />
          </div>
          <div className="flex1 u-border--gray">
            {this.renderEditor(POD_LOGS, "text")}
          </div>
        </div>
      )
    case "podEvents":
      return (
        <div className="flex1 u-border--gray">
          {this.renderEditor(this.state.podEvents, "yaml")}
        </div>
      )
    default:
      return <div>nothing selected</div>
    }
  }

  renderEditor = (content, mode) => {
    if (this.state.loading) {
      return (
        <div style={{ height: 500 }} className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      );
    }

    return (
      <AceEditor
        ref={el => (this.aceEditor = el)}
        mode={mode}
        theme="chrome"
        className="flex1 flex"
        readOnly={true}
        value={content}
        height="500px"
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
    )
  }

  render() {
    const { pod } = this.props;
    return (
        <div className="flex1 flex-column">
          <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--more">Details for {pod.primary}</p>
          <div className="SupportBundleTabs--wrapper flex-column flex1">
            <div className="flex tab-items">
              <span className={`${this.state.activeTab === "podDefinition" ? "is-active" : ""} tab-item blue`} onClick={() => this.togglePodDetailView("podDefinition")}>Pod definition</span>
              <span className={`${this.state.activeTab === "podLogs" ? "is-active" : ""} tab-item blue`} onClick={() => this.togglePodDetailView("podLogs")}>Pod logs</span>
              <span className={`${this.state.activeTab === "podEvents" ? "is-active" : ""} tab-item blue`} onClick={() => this.togglePodDetailView("podEvents")}>Pod events</span>
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

export default withRouter(PodAnalyzerDetails);
