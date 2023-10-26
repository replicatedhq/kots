import classNames from "classnames";

export default function SidebarLayout(props) {
  const { className, children, sidebar, condition = true } = props;

  return (
    <div className={classNames(className)}>
      {Boolean(condition) && sidebar}
      {children}
    </div>
  );
}
