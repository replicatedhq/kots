import { Schema } from "jsonschema";

export const schema: Schema = {
  type: "object",
  properties: {
    v1: {
      type: "object",
      properties: {
        ChartRepoURL: {
          type: "string",
        },
        ChartVersion: {
          type: "string",
        },
        chartURL: {
          type: "string",
        },
        config: {
          type: ["object", "null"],
          additionalProperties: true,
        },
        contentSHA: {
          type: "string",
        },
        helmValues: {
          type: "string",
        },
        helmValuesDefaults: {
          type: "string",
        },
        kustomize: {
          type: "object",
          properties: {
            overlays: {
              type: "object",
              properties: {
                ".*": {
                  type: "object",
                },
              },
            },
          },
        },
        lifecycle: {
          type: "object",
          properties: {
            stepsCompleted: {
              type: "object",
              additionalProperties: true,
            },
          },
        },
        metadata: {
          type: ["object", "null"],
          properties: {
            ".*": {
              type: "string",
            },
          },
        },
        terraform: {},
        upstream: {
          type: "string",
        },
      },
      required: ["config", "metadata"],
    },
  },
  required: ["v1"],
};
