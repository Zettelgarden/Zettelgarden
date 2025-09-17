import React from 'react';
import { HelpContent as HelpContentType } from '../../types/help';

interface HelpContentProps {
  content: HelpContentType;
}

export function HelpContent({ content }: HelpContentProps) {
  switch (content.type) {
    case 'text':
      return <p className="text-gray-700">{content.data}</p>;

    case 'list':
      return (
        <ul className="list-disc ml-6 space-y-2">
          {content.data.map((item: string, index: number) => (
            <li key={index} className="text-gray-700">{item}</li>
          ))}
        </ul>
      );

    case 'code':
      return (
        <pre className="bg-gray-100 p-4 rounded-lg overflow-x-auto">
          <code className="text-sm">{content.data}</code>
        </pre>
      );

    case 'callout':
      const { type = 'info', title, message } = content.data;
      const calloutStyles = {
        info: 'bg-blue-50 border-blue-200 text-blue-800',
        warning: 'bg-yellow-50 border-yellow-200 text-yellow-800',
        success: 'bg-green-50 border-green-200 text-green-800',
        error: 'bg-red-50 border-red-200 text-red-800'
      };

      return (
        <div className={`p-4 rounded-lg border ${calloutStyles[type as keyof typeof calloutStyles]}`}>
          {title && <h4 className="font-semibold mb-2">{title}</h4>}
          <p>{message}</p>
        </div>
      );

    case 'video':
      return (
        <div className="aspect-video bg-gray-100 rounded-lg flex items-center justify-center">
          <span className="text-gray-500">Video: {content.data.title}</span>
        </div>
      );

    case 'interactive':
      return (
        <div className="p-4 bg-gray-50 rounded-lg border-2 border-dashed border-gray-300">
          <span className="text-gray-500">Interactive Demo: {content.data.title}</span>
        </div>
      );

    default:
      return null;
  }
}