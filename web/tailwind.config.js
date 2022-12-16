/* eslint-disable @typescript-eslint/no-var-requires */
/** @type {import('tailwindcss').Config} */
const plugin = require("tailwindcss/plugin");
const defaultTheme = require("tailwindcss/defaultTheme");

module.exports = {
  prefix: "tw-",
  content: ["./src/index.html", "./src/**/*.{js,jsx,ts,tsx}"],
  theme: {
    colors: {
      white: "#ffffff",
      teal: "#4db9c0",
      "teal-muted-dark": "#577981",
      "teal-medium": "#097992",
      gray: {
        100: "#dfdfdf",
        200: "#c4c8ca",
        300: "#b3b3b3",
        400: "#959595",
        500: "#717171",
        600: "#585858",
        700: "#4f4f4f",
        800: "#323232"
      },
      neutral: {
        700: "#4A4A4A"
      },
      error: "#bc4752",
      "error-xlight": "#fbedeb",
      "error-dark": "#98222d",
      "error-bright": "#f65C5C",
      "success-bright": "#38cc97",
      disabled: "#9c9c9c",
      "warning-bright": "#ec8f39",
      "info-bright": "#76bbca",
      "disabled-teal": "#76a6cf",
      "dark-neon-green": "#38cc97"
    },
    extend: {
      borderRadius: {
        xs: "0.125rem",
        sm: "0.187rem",
        variants: ["first", "last"]
      },
      fontFamily: {
        sans: ["Open Sans", ...defaultTheme.fontFamily.sans]
      }
    }
  },
  corePlugins: {
    preflight: false
  },
  plugins: [
    plugin(function ({ addVariant }) {
      addVariant("is-enabled", "&:not([disabled])");
      addVariant("is-disabled", "&[disabled]");
    })
  ]
};
