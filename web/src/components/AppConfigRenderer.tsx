import ConfigRender from "./config_render/ConfigRender";
import map from "lodash/map";
import sortBy from "lodash/sortBy";
import keyBy from "lodash/keyBy";

type ConfigGroupItem = {
  default: string;
  error: string;
  hidden: boolean;
  name: string;
  required: boolean;
  title: string;
  type: string;
  validationError: string;
  value: string;
  when: "true" | "false";
};

type ConfigGroup = {
  hidden: boolean;
  hasError: boolean;
  items: ConfigGroupItem[];
  name: string;
  title: string;
  when: "true" | "false";
};

interface AppConfigRendererProps {
  appSlug: string;
  configSequence: string;
  getData: (group: ConfigGroup[]) => void;
  groups: ConfigGroup[];
  handleChange?: () => void;
  handleDownloadFile: (filename: string) => void;
  readonly?: boolean;
}

export const AppConfigRenderer = ({
  groups,
  handleChange,
  getData,
  handleDownloadFile,
  readonly = false,
  configSequence,
  appSlug,
}: AppConfigRendererProps) => {
  const orderedFields = sortBy(groups, "position");
  const _groups = keyBy(orderedFields, "name");
  const groupsList = map(groups, "name");

  return (
    <div id="config-render-component">
      <ConfigRender
        fieldsList={groupsList}
        fields={_groups}
        rawGroups={orderedFields}
        handleChange={
          handleChange ||
          (() => {
            return;
          })
        }
        getData={
          getData ||
          (() => {
            return;
          })
        }
        handleDownloadFile={
          handleDownloadFile ||
          (() => {
            return;
          })
        }
        readonly={readonly}
        configSequence={configSequence}
        appSlug={appSlug}
      />
    </div>
  );
};
