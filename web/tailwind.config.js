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
        100: "#dedede",
        200: "#c4c8ca",
        300: "#b3b3b3",
        410: "#9b9b9b",
        400: "#959595",
        500: "#717171",
        600: "#585858",
        700: "#4f4f4f",
        800: "#323232",
        900: "#2c2c2c",
      },
      blue: {
        50: "#ecf4fe",
        75: "#b3d2fc",
        200: "#65a4f8",
        300: "#4591f7",
        400: "#3066ad",
      },
      green: {
        50: "#e7f7f3",
        75: "#9cdfcf",
        100: "#73d2bb",
        200: "#37bf9e",
        300: "#0eb28a",
        400: "#0a7d61",
        500: "#096d54",
      },
      indigo: {
        100: "#f0f1ff",
        200: "#c2c7fd",
        300: "#a9b0fd",
        400: "#838efc",
        500: "#6a77fb",
        600: "#4a53b0",
        700: "#414999",
      },
      neutral: {
        700: "#4A4A4A",
      },
      teal: {
        300: "#4db9c0",
        400: "#38a3a8",
      },
      pink: {
        50: "#fff0f3",
        100: "#ffc1cf",
        200: "#fea7bc",
        300: "#fe819f",
        400: "#fe678b",
        500: "#b24861",
        600: "#9b3f55",
      },
      purple: {
        400: "#7242b0",
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
      "dark-neon-green": "#38cc97",
    },
    extend: {
      borderRadius: {
        xs: "0.125rem",
        sm: "0.187rem",
        variants: ["first", "last"],
      },
      fontFamily: {
        sans: ["Open Sans", ...defaultTheme.fontFamily.sans],
      },
    },
  },
  corePlugins: {
    preflight: false,
  },
  plugins: [
    plugin(function ({ addVariant }) {
      addVariant("is-enabled", "&:not([disabled])");
      addVariant("is-disabled", "&[disabled]");
    }),
  ],
};
