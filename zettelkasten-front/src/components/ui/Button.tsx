import React from 'react';

export interface ButtonProps {
  onClick?: () => void;
  disabled?: boolean;
  type?: 'button' | 'submit' | 'reset';
  'aria-pressed'?: boolean;
  variant?: 'primary' | 'secondary' | 'outline' | 'ghost' | 'danger';
  size?: 'small' | 'medium' | 'large';
  children: React.ReactNode;
  className?: string;
  title?: string;
}

export const Button: React.FC<ButtonProps> = ({
  onClick,
  disabled = false,
  type,
  'aria-pressed': ariaPressed,
  variant = 'primary',
  size = 'medium',
  children,
  className = '',
  title,
}) => {
  const baseClasses =
    'font-semibold rounded focus:outline-none focus:ring-2 focus:ring-offset-2';
  const variantClasses = {
    primary:
      'bg-palette-dark text-white hover:bg-palette-darkest focus:ring-blue-500',
    secondary:
      'bg-gray-200 text-gray-800 hover:bg-gray-300 focus:ring-gray-500',
    outline:
      'bg-transparent border border-gray-300 text-gray-700 hover:bg-gray-50 focus:ring-gray-500',
    // Transparent background, no text color of its own: consumers own the
    // text/hover colors so icon & secondary-header buttons keep their exact
    // look. Note: don't add text-* here — it would win over consumer
    // text-gray-400/500 overrides (gray-700 sorts after 400/500 in Tailwind).
    ghost: 'bg-transparent hover:bg-gray-100 focus:ring-gray-500',
    danger: 'bg-red-600 text-white hover:bg-red-700 focus:ring-red-500',
  };
  const sizeClasses = {
    small: 'px-3 py-2 md:py-1 min-h-[44px] md:min-h-[32px] text-sm',
    medium: 'px-4 py-3 min-h-[44px]',
    large: 'px-6 py-3 min-h-[44px] text-lg',
  };

  const classes = `${baseClasses} ${variantClasses[variant]} ${sizeClasses[size]} ${className}`;

  return (
    <button
      onClick={onClick}
      disabled={disabled}
      type={type}
      aria-pressed={ariaPressed}
      className={classes}
      title={title}
    >
      {children}
    </button>
  );
};
