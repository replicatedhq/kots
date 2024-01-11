import { ComponentStory, ComponentMeta } from "@storybook/react";

import CodeSnippet from "../components/shared/CodeSnippet";

export default {
  title: "Example/CodeSnippet",
  component: CodeSnippet,
} as ComponentMeta<typeof CodeSnippet>;

const Template: ComponentStory<typeof CodeSnippet> = () => (
  <CodeSnippet
    language="bash"
    canCopy={true}
    onCopyText={
      <span className="u-textColor--success">
        Command has been copied to your clipboard
      </span>
    }
  >
    Copy me!
  </CodeSnippet>
);

export const CodeSnippetExample = Template.bind({});
