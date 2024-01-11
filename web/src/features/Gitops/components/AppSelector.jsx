import Select from "react-select";
import { getLabel } from "../utils";

const AppSelector = ({ apps, selectedApp, handleAppChange, isSingleApp }) => {
  return (
    <div className="flex flex1 flex-column u-marginRight--10">
      <p className="card-item-title">Select an application to configure</p>
      <div className="u-position--relative u-marginTop--5 u-marginBottom--10">
        <Select
          className="replicated-select-container select-large "
          classNamePrefix="replicated-select"
          placeholder="Select an application"
          options={apps}
          isSearchable={false}
          getOptionLabel={(app) => getLabel(app, isSingleApp)}
          value={selectedApp}
          onChange={handleAppChange}
          isOptionSelected={(option) => {
            option.value === selectedApp;
          }}
        />
      </div>
    </div>
  );
};

export default AppSelector;
