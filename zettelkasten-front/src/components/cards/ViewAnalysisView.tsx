import React from "react";
import { SectionAnalysis } from "../../api/summarizer";

interface ViewAnalysisViewProps {
  analysis: SectionAnalysis[] | null;
}

export function ViewAnalysisView({ analysis }: ViewAnalysisViewProps) {
  if (!analysis || analysis.length === 0) {
    return (
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-6">
        <div className="text-blue-800 text-center">
          <p className="font-medium">No analysis available</p>
          <p className="text-sm mt-2">Generate an analysis to see it here.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-blue-50 border border-blue-200 rounded-lg p-6 shadow-sm">
      <div className="bg-blue-100 text-blue-800 font-semibold px-4 py-2 rounded-md mb-4">
        Analysis View
      </div>
      <div className="space-y-4">
        {analysis.map((section, index) => (
          <div key={index} className="bg-white rounded-md p-4">
            <h2 className="font-bold text-lg text-gray-800 mb-3">{section.section}</h2>
            {section.theses && section.theses.map((thesis, thesisIndex) => (
              <div key={thesisIndex} className="mb-4 last:mb-0">
                <div className="flex items-start gap-2">
                  <span className="text-gray-700 font-medium">{thesis.thesis}</span>
                </div>
                {thesis.arguments && thesis.arguments.length > 0 && (
                  <div className="ml-4 mt-2">
                    <ul className="list-disc ml-5 space-y-1">
                      {thesis.arguments.map((arg, argIndex) => (
                        <li key={argIndex} className="text-gray-600 text-sm">
                          {arg.argument}
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
                {thesisIndex < (section.theses?.length || 0) - 1 && (
                  <hr className="mt-3 border-gray-200" />
                )}
              </div>
            ))}
          </div>
        ))}
      </div>
    </div>
  );
}
