import { useQuery } from "@tanstack/react-query";
import cx from "classnames";
import React, { ChangeEvent, useState } from "react";
import Modal from "react-modal";

import Icon from "@components/Icon";
import CodeSnippet from "@components/shared/CodeSnippet";
import { Utilities } from "@src/utilities/utilities";

const AddNodeModal = ({
  showModal,
  handleCloseModal,
}: {
  showModal: boolean;
  handleCloseModal: () => void;
}) => {
  const [selectedNodeTypes, setSelectedNodeTypes] = useState<string[]>([]);

  type AddNodeCommandResponse = {
    command: string;
    expiry: string;
  };

  const {
    data: generateAddNodeCommand,
    isLoading: generateAddNodeCommandLoading,
    error: generateAddNodeCommandError,
  } = useQuery<AddNodeCommandResponse, Error, AddNodeCommandResponse>({
    queryKey: ["generateAddNodeCommand", selectedNodeTypes],
    queryFn: async ({ queryKey }) => {
      const [, nodeTypes] = queryKey;
      const res = await fetch(
        `${process.env.API_ENDPOINT}/helmvm/generate-node-join-command`,
        {
          headers: {
            "Content-Type": "application/json",
            Accept: "application/json",
          },
          credentials: "include",
          method: "POST",
          body: JSON.stringify({
            roles: nodeTypes,
          }),
        }
      );
      if (!res.ok) {
        if (res.status === 401) {
          Utilities.logoutUser();
        }
        console.log(
          "failed to get generate node command, unexpected status code",
          res.status
        );
        try {
          const error = await res.json();
          throw new Error(
            error?.error?.message || error?.error || error?.message
          );
        } catch (err) {
          throw new Error(
            "Unable to generate node join command, please try again later."
          );
        }
      }
      return res.json();
    },
    enabled: selectedNodeTypes.length > 0,
  });
  // #region node type logic
  const NODE_TYPES = ["controller"];

  const determineDisabledState = () => {
    return false;
  };

  const handleSelectNodeType = (e: ChangeEvent<HTMLInputElement>) => {
    let nodeType = e.currentTarget.value;
    let types = selectedNodeTypes;

    if (selectedNodeTypes.includes(nodeType)) {
      setSelectedNodeTypes(types.filter((type) => type !== nodeType));
    } else {
      setSelectedNodeTypes([...types, nodeType]);
    }
  };
  // #endregion
  return (
    <Modal
      isOpen={showModal}
      onRequestClose={handleCloseModal}
      contentLabel="Add Node"
      className="Modal"
      ariaHideApp={false}
    >
      <div className="Modal-body tw-flex tw-flex-col tw-gap-4 tw-font-sans">
        <div className="tw-flex">
          <h1 className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--more">
            Add a Node
          </h1>
          <Icon
            icon="close"
            size={14}
            className="tw-ml-auto gray-color clickable close-icon"
            onClick={handleCloseModal}
          />
        </div>
        <p className="tw-text-base tw-text-gray-600">
          To add a node to this cluster, select the type of node you'd like to
          add. Once you've selected a node type, we will generate a node join
          command for you to use in the CLI. When the node successfully joins
          the cluster, you will see it appear in the list of nodes on this page.
        </p>
        <div className="tw-grid tw-gap-2 tw-grid-cols-4 tw-auto-rows-auto">
          {NODE_TYPES.map((nodeType) => (
            <div
              key={nodeType}
              className={cx("BoxedCheckbox", {
                "is-active": selectedNodeTypes.includes(nodeType),
                "is-disabled": determineDisabledState(),
              })}
            >
              <input
                id={`${nodeType}NodeType`}
                className="u-cursor--pointer hidden-input"
                type="checkbox"
                name={`${nodeType}NodeType`}
                value={nodeType}
                disabled={determineDisabledState()}
                checked={selectedNodeTypes.includes(nodeType)}
                onChange={handleSelectNodeType}
              />
              <label
                htmlFor={`${nodeType}NodeType`}
                className="tw-block u-cursor--pointer u-userSelect--none u-textColor--primary u-fontSize--normal u-fontWeight--medium tw-text-center"
              >
                {nodeType === "controller" ? "controlplane" : nodeType}
              </label>
            </div>
          ))}
        </div>
        <div>
          {selectedNodeTypes.length > 0 && generateAddNodeCommandLoading && (
            <p className="tw-text-base tw-w-full tw-text-center tw-py-4 tw-text-gray-500 tw-font-semibold">
              Generating command...
            </p>
          )}
          {!generateAddNodeCommand && generateAddNodeCommandError && (
            <p className="tw-text-base tw-w-full tw-text-center tw-py-4 tw-text-pink-500 tw-font-semibold">
              {generateAddNodeCommandError?.message}
            </p>
          )}
          {!generateAddNodeCommandLoading && generateAddNodeCommand?.command && (
            <>
              <CodeSnippet
                key={selectedNodeTypes.toString()}
                language="bash"
                canCopy={true}
                onCopyText={
                  <span className="u-textColor--success">Copied!</span>
                }
              >
                {generateAddNodeCommand?.command}
              </CodeSnippet>
              <p className="tw-text-sm tw-text-gray-500 tw-font-semibold tw-mt-2">
                Command expires: {generateAddNodeCommand?.expiry}
              </p>
            </>
          )}
        </div>
        {/* buttons */}
        <div className="tw-w-full tw-flex tw-justify-end tw-gap-2">
          <button className="btn secondary large" onClick={handleCloseModal}>
            Close
          </button>
        </div>
      </div>
    </Modal>
  );
};

export default AddNodeModal;
