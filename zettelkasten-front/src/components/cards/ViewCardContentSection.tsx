import React from "react";
import ReactMarkdown from "react-markdown";
import { TaskListItem } from "../tasks/TaskListItem";
import { Card } from "../../models/Card";
import { SummarizeJobResponse } from "../../api/summarizer";
import { HeaderSubSection } from "../Header";
import { CardBody } from "./CardBody";

interface ViewCardContentSectionProps {
  viewingCard: Card;
  showingSummary?: boolean;
  latestSummary: SummarizeJobResponse | null;
  onSaveCard?: (updatedCard: Card) => void | Promise<void>;
}

export function ViewCardContentSection({
  viewingCard,
  showingSummary = false,
  latestSummary,
  onSaveCard,
}: ViewCardContentSectionProps) {
  return (
    <div className="space-y-8">
      <div
        className={`prose prose-sm max-w-none ${
          showingSummary
            ? "bg-yellow-50 border border-yellow-200 rounded-lg px-4 py-3"
            : ""
        }`}
      >
        {showingSummary && latestSummary?.result ? (
          <div>
            <div className="bg-yellow-100 text-yellow-800 font-semibold text-sm px-3 py-2 rounded-md mb-4">
              Summary View
            </div>
            <div className="prose prose-sm">
              <ReactMarkdown>{latestSummary.result}</ReactMarkdown>
            </div>
          </div>
        ) : (
          <CardBody
            viewingCard={viewingCard}
            entities={viewingCard.entities}
            onSave={onSaveCard}
          />
        )}
      </div>

      {/* Tasks Section */}
      {viewingCard.tasks.length > 0 && (
        <div>
          <HeaderSubSection text="Tasks" />
          <div className="mt-2 space-y-2">
            {viewingCard.tasks.map((task, index) => (
              <TaskListItem
                key={task.id}
                task={task}
                onTagClick={(tag: string) => {}}
              />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
