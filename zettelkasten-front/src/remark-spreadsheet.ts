import { visit } from "unist-util-visit";
import { Node } from "unist";

export default function remarkSpreadsheet() {
  return (tree: Node) => {
    visit(tree, "text", (node: any, index: number | undefined, parent: any) => {
      if (!parent || typeof node.value !== "string" || index === undefined) return;

      // Match {{spreadsheet:123}} or {{spreadsheet}} syntax (numeric ID)
      const regex = /\{\{spreadsheet(?::(\d+))?\}\}/gi;
      let match;
      const newNodes: any[] = [];
      let lastIndex = 0;

      while ((match = regex.exec(node.value)) !== null) {
        const [fullMatch, id] = match;
        const spreadsheetId = id || "";

        // Push text before match
        if (match.index > lastIndex) {
          newNodes.push({
            type: "text",
            value: node.value.slice(lastIndex, match.index),
          });
        }

        // Push spreadsheet node
        newNodes.push({
          type: "spreadsheet",
          data: {
            id: spreadsheetId,
            hName: "div",
            hProperties: {
              className: "spreadsheet-container",
              "data-spreadsheet-id": spreadsheetId
            }
          },
          children: [],
        });

        lastIndex = match.index + fullMatch.length;
      }

      // Push remaining text after last match
      if (lastIndex < node.value.length) {
        newNodes.push({
          type: "text",
          value: node.value.slice(lastIndex),
        });
      }

      if (newNodes.length > 0) {
        parent.children.splice(index, 1, ...newNodes);
        return index + newNodes.length;
      }
    });
  };
}
