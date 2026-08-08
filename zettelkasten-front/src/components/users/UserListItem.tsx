import React from 'react';
import { Link } from 'react-router-dom';
import { User } from '../../models/User';

interface UserListItemProps {
  user: User;
}

export function UserListItem({ user }: UserListItemProps) {
  return (
    <tr className="border-b hover:bg-gray-100">
      <td className="py-2 px-4">{user.id}</td>
      <td className="py-2 px-4">
        <Link
          to={`/admin/user/${user.id}`}
          className="text-blue-600 hover:text-blue-800"
        >
          {user.username}
        </Link>
      </td>
      <td className="py-2 px-4">
        <span
          className={`px-2 py-1 rounded text-sm ${
            user.is_admin
              ? 'bg-purple-100 text-purple-800'
              : 'bg-gray-100 text-gray-800'
          }`}
        >
          {user.is_admin ? 'Yes' : 'No'}
        </span>
      </td>
      <td className="py-2 px-4">
        {new Date(user.last_login).toLocaleString()}
      </td>
      <td className="py-2 px-4">{user.email}</td>
      <td className="py-2 px-4">
        <span
          className={`px-2 py-1 rounded text-sm ${
            user.email_validated
              ? 'bg-green-100 text-green-800'
              : 'bg-yellow-100 text-yellow-800'
          }`}
        >
          {user.email_validated ? 'Verified' : 'Pending'}
        </span>
      </td>
      <td className="py-2 px-4">
        <span
          className={`px-2 py-1 rounded text-sm ${
            user.stripe_subscription_status === 'active'
              ? 'bg-green-100 text-green-800'
              : user.stripe_subscription_status === 'trialing'
              ? 'bg-blue-100 text-blue-800'
              : 'bg-red-100 text-red-800'
          }`}
        >
          {user.stripe_subscription_status}
        </span>
      </td>
      <td className="py-2 px-4">
        {new Date(user.created_at).toLocaleString()}
      </td>
      <td className="py-2 px-4">{user.card_count}</td>
    </tr>
  );
}
