import React, { ReactNode } from 'react';
import { useNavigate, Link } from 'react-router-dom';

interface AdminTopBarProps {
  children?: ReactNode;
}

export function AdminTopBar({ children }: AdminTopBarProps) {
  return (
    <div className="flex bg-white w-full h-[50px] items-center justify-between">
      <div className="pl-[10px]">
        <div>
          <h1>
            <Link to="/admin">Zettelindex Admin</Link>
          </h1>
        </div>
      </div>
      <div className="flex justify-end items-center">
        <button className="px-5 py-[10px] bg-blue-600 text-white no-underline border-none rounded-lg cursor-pointer text-base inline-block mx-1 hover:bg-blue-700 transition-colors duration-300 focus:outline-none">
          <Link to="/app">Back To App</Link>
        </button>
      </div>
    </div>
  );
}

export default AdminTopBar;
