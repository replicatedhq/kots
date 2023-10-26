import "../../scss/components/shared/Warning.scss";

const Warning = ({ children }) => {
  return (
    <div className="warning-container flex">
      <span className="icon errorWarningIcon" />
      <p className="warning-text">{children}</p>
    </div>
  );
};

export default Warning;
