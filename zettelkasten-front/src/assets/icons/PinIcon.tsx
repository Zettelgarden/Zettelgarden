import React from "react";

interface PinIconProps {
  className?: string;
  filled?: boolean;
}

export function PinIcon({ className = "", filled = false }: PinIconProps) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill={filled ? "currentColor" : "none"}
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
    >
      <path d="M9 9V4.5a2.5 2.5 0 0 1 5 0V9a2 2 0 0 1 2 2v0.5a1 1 0 0 1-1 1h-5.5l.5 8v0a1 1 0 0 1-1 1v0a1 1 0 0 1-1-1v0l.5-8H4a1 1 0 0 1-1-1v-0.5a2 2 0 0 1 2-2Z"/>
    </svg>
  );
}