import React from "react";
import { ThemeContext } from "@src/Root";

export default function withTheme(Component) {
  return function withThemeComponent(props) {
    return (
      <ThemeContext.Consumer>
        {values => <Component {...values} {...props} />}
      </ThemeContext.Consumer>
    );
  }
}
