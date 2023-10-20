import classNames from "classnames";

import "@src/scss/components/shared/PaperIcon.scss";

export default function PaperIcon(props) {
  const { iconClass, className, onClick, height, width } = props;

  return (
    <div
      className={classNames(
        "PaperIcon flex alignItems--center justifyContent--center",
        { clickable: onClick },
        className
      )}
      onClick={onClick}
      style={{ height, width }}
    >
      <span className={classNames("icon", iconClass, { clickable: onClick })} />
    </div>
  );
}

PaperIcon.defaultProps = {
  height: "25px",
  width: "25px",
};
