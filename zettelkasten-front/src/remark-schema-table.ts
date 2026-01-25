import { visit } from "unist-util-visit";
import { Node } from "unist";

export default function remarkSchemaTable() {
    return (tree: Node) => {
        visit(tree, "text", (node: any, index: number | undefined, parent: any) => {
            if (!parent || typeof node.value !== "string" || index === undefined) return;

            // Match &SCHEMATABLE:schemaId& or &SCHEMATABLE:schemaId|columns& or &SCHEMATABLE:schemaId|columns|filters&
            const regex = /&SCHEMATABLE:([^&|]+)(?:\|([^&|]*))?(?:\|([^&]+))?&/g;
            let match;
            const newNodes: any[] = [];
            let lastIndex = 0;

            while ((match = regex.exec(node.value)) !== null) {
                const [fullMatch, schemaId, columns, filters] = match;

                // Push text before match
                if (match.index > lastIndex) {
                    newNodes.push({
                        type: "text",
                        value: node.value.slice(lastIndex, match.index),
                    });
                }

                // Push schema table node with optional columns and filters data attributes
                const hProperties: any = {
                    className: "schema-table-container",
                    "data-schema-id": schemaId
                };

                if (columns) {
                    hProperties["data-columns"] = columns;
                }

                if (filters) {
                    // Decode the filters
                    const decodedFilters = filters.replace(/%26/g, '&').replace(/%7C/g, '|');
                    hProperties["data-filters"] = decodedFilters;
                }

                newNodes.push({
                    type: "schemaTable",
                    data: {
                        schemaId,
                        columns,
                        filters: filters ? filters.replace(/%26/g, '&').replace(/%7C/g, '|') : undefined,
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
