import { Component } from "react";

export default class MountAware extends Component {
  componentDidMount() {
    if (this.props.onMount) {
      const element = document.getElementById("mount-aware-wrapper");
      this.props.onMount(element);
    }
  }

  render() {
    const { children, className, dataTestId } = this.props;

    return (
      <div
        id="mount-aware-wrapper"
        className={className}
        data-testid={dataTestId}
      >
        {children}
      </div>
    );
  }
}
