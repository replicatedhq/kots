import Icon from "@src/components/Icon";

const YamlErrors = ({ yamlErrors, handleShowDetailsClicked }) => {
  return (
    <div className="flex alignItems--center u-marginTop--5">
      <Icon icon={"warning-circle-filled"} size={16} className="error-color" />
      <span className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginLeft--5 u-textColor--error">
        {yamlErrors?.length} Invalid file
        {yamlErrors?.length !== 1 ? "s" : ""}{" "}
      </span>
      <span
        className="link u-marginLeft--5 u-fontSize--small"
        onClick={handleShowDetailsClicked}
      >
        {" "}
        See details{" "}
      </span>
    </div>
  );
};

export { YamlErrors };
