import React from "react";

interface ButtonProps {
  onClick?: () => void;
  disabled?: boolean;
  variant?: "primary" | "secondary" | "outline";
  size?: "small" | "medium" | "large";
  children: React.ReactNode;
  className?: string;
  title?: string;
}

export const Button: React.FC<ButtonProps> = ({
  onClick,
  disabled = false,
  variant = "primary",
  size = "medium",
  children,
  className = "",
  title,
}) => {
  const baseClasses =
    "font-semibold rounded focus:outline-none focus:ring-2 focus:ring-offset-2";
  const variantClasses = {
    primary:
      "bg-palette-dark text-white hover:bg-palette-darkest focus:ring-blue-500",
    secondary:
      "bg-gray-200 text-gray-800 hover:bg-gray-300 focus:ring-gray-500",
    outline:
      "bg-transparent border border-gray-300 text-gray-700 hover:bg-gray-50 focus:ring-gray-500",
  };
  const sizeClasses = {
    small: "px-3 py-2 md:py-1 min-h-[44px] md:min-h-[32px] text-sm",
    medium: "px-4 py-3 min-h-[44px]",
    large: "px-6 py-3 min-h-[44px] text-lg",
  };

  const classes = `${baseClasses} ${variantClasses[variant]} ${sizeClasses[size]} ${className}`;

  return (
    <button onClick={onClick} disabled={disabled} className={classes} title={title}>
      {children}
    </button>
  );
};
