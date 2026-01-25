import { visit } from "unist-util-visit";
import { Node } from "unist";

export default function remarkSchemaTable() {
    return (tree: Node) => {
        visit(tree, "text", (node: any, index: number | undefined, parent: any) => {
            if (!parent || typeof node.value !== "string" || index === undefined) return;

            // Match both old and new schema table syntax:
            // Old: &SCHEMATABLE:schemaId& or &SCHEMATABLE:schemaId|col1,col2&
            // New: {{schema:slug}} or {{schema:slug|col1,col2}}
            const regex = /(?:&SCHEMATABLE:|{{schema:)([^&|}\s]+)(?:\|([^&}]+?))?(?:&|}})/g;
            let match;
            const newNodes: any[] = [];
            let lastIndex = 0;

            while ((match = regex.exec(node.value)) !== null) {
                const [fullMatch, schemaRef, columns] = match;

                // Push text before match
                if (match.index > lastIndex) {
                    newNodes.push({
                        type: "text",
                        value: node.value.slice(lastIndex, match.index),
                    });
                }

                // Push schema table node with optional columns data attribute
                // schemaRef can be either an ID (old format) or a slug (new format)
                const hProperties: any = {
                    className: "schema-table-container",
                    "data-schema-ref": schemaRef
                };

                if (columns) {
                    hProperties["data-columns"] = columns;
                }

                newNodes.push({
                    type: "schemaTable",
                    data: {
                        schemaRef,
                        columns,
                        hName: "div",
                        hProperties
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
