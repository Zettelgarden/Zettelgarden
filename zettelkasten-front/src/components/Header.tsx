import React from "react";

interface HeaderProps {
  text: string;
  className?: string;
}
interface HeadlineProps {
  children: string;
  className?: string;
}

export function HeaderTop({ text, className }: HeaderProps) {
  return <span className={`font-bold text-lg ${className || ""}`}>{text}</span>;
}

export function HeaderSection({ text, className }: HeaderProps) {
  return <span className={`font-bold text-base ${className || ""}`}>{text}</span>;
}

export function HeaderSubSection({ text, className }: HeaderProps) {
  return (
    <div>
      <span className={`font-bold text-xs ${className || ""}`}>{text}</span>
      <br />
    </div>
  );
}

export function H1({ children, className }: HeadlineProps) {
  return <h1 className={`text-xl font-bold ${className || ""}`}>{children}</h1>;
}

export function H2({ children, className }: HeadlineProps) {
  return <h2 className={`text-lg font-bold ${className || ""}`}>{children}</h2>;
}

export function H3({ children, className }: HeadlineProps) {
  return <h3 className={`text-base font-bold ${className || ""}`}>{children}</h3>;
}

export function H4({ children, className }: HeadlineProps) {
  return <h4 className={`text-sm font-bold ${className || ""}`}>{children}</h4>;
}

export function H5({ children, className }: HeadlineProps) {
  return <h5 className={`text-xs font-bold ${className || ""}`}>{children}</h5>;
}

export function H6({ children, className }: HeadlineProps) {
  return <h6 className={`text-xs font-bold ${className || ""}`}>{children}</h6>;
}