export default function Tab(props) {
  const { className, children } = props;

  return <div className={className}>{children}</div>;
}
