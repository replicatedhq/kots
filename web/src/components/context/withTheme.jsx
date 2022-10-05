import React from "react";
import { ThemeContext } from "@src/Root";

export default function withTheme(Component) {
  return function withThemeComponent(props) {
    return (
      <ThemeContext.Consumer>
        {(values) => <Component {...values} {...props} />}
      </ThemeContext.Consumer>
    );
  };
}

function useTheme() {
  const context = React.useContext(ThemeContext);
  if (context === undefined) {
    throw new Error("useTheme must be used within a ThemeProvider");
  }
  return context;
}

export { useTheme };
