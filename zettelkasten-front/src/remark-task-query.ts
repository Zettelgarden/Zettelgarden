import { visit } from 'unist-util-visit';
import { Node } from 'unist';

export default function remarkTaskQuery() {
  return (tree: Node) => {
    visit(tree, 'text', (node: any, index: number | undefined, parent: any) => {
      if (!parent || typeof node.value !== 'string' || index === undefined)
        return;

      const regex = /&TASKQUERY:([^&]*)&/g;
      let match;
      const newNodes: any[] = [];
      let lastIndex = 0;

      while ((match = regex.exec(node.value)) !== null) {
        const [fullMatch, query] = match;

        // Push text before match
        if (match.index > lastIndex) {
          newNodes.push({
            type: 'text',
            value: node.value.slice(lastIndex, match.index),
          });
        }

        // Push task query node
        newNodes.push({
          type: 'taskQuery',
          data: {
            query,
            hName: 'div',
            hProperties: {
              className: 'task-query-container',
              'data-query': query,
            },
          },
          children: [],
        });

        lastIndex = match.index + fullMatch.length;
      }

      // Push remaining text after last match
      if (lastIndex < node.value.length) {
        newNodes.push({
          type: 'text',
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
