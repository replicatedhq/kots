import { ThemeContext } from "@src/Root";
import { useContext } from "react";

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
  const context = useContext(ThemeContext);
  if (context === undefined) {
    throw new Error("useTheme must be used within a ThemeProvider");
  }
  return context;
}

export { useTheme };
