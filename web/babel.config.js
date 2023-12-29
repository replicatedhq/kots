module.exports = {
  presets: [
    "@babel/preset-env",
    ["@babel/preset-react", { runtime: "automatic" }],
    "@babel/preset-typescript",
  ],
  plugins: [
    "@babel/plugin-proposal-class-properties",
    "@babel/plugin-proposal-optional-chaining",
  ],
  "assumptions": {
    "privateFieldsAsProperties": true,
    "setPublicClassFields": true
  }
};
