import React from 'react';

// Table components
export const Table: React.FC<React.HTMLAttributes<HTMLTableElement>> = ({
  children,
  ...props
}) => (
  <table className="min-w-full border-collapse my-4" {...props}>
    {children}
  </table>
);

export const TableHead: React.FC<
  React.HTMLAttributes<HTMLTableSectionElement>
> = ({ children, ...props }) => (
  <thead className="bg-gray-100" {...props}>
    {children}
  </thead>
);

export const TableBody: React.FC<
  React.HTMLAttributes<HTMLTableSectionElement>
> = ({ children, ...props }) => <tbody {...props}>{children}</tbody>;

export const TableRow: React.FC<React.HTMLAttributes<HTMLTableRowElement>> = ({
  children,
  ...props
}) => (
  <tr className="border-b" {...props}>
    {children}
  </tr>
);

export const TableHeader: React.FC<
  React.TdHTMLAttributes<HTMLTableCellElement>
> = ({ children, ...props }) => (
  <th className="py-2 px-4 font-bold text-left border" {...props}>
    {children}
  </th>
);

export const TableCell: React.FC<
  React.TdHTMLAttributes<HTMLTableCellElement>
> = ({ children, ...props }) => (
  <td className="py-2 px-4 border" {...props}>
    {children}
  </td>
);
