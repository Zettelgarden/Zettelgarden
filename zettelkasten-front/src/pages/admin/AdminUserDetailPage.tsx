import React, { useState, useEffect } from 'react';
import { requestPasswordReset } from '../../api/auth';
import { adminDeleteUser } from '../../api/account';
import { getUser } from '../../api/users';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { User } from '../../models/User';

export function AdminUserDetailPage() {
  const [user, setUser] = useState<User | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  function handlePasswordReset() {
    if (user === null) return;
    requestPasswordReset(user.email);
  }

  async function handleDelete() {
    if (user === null) return;
    if (
      !window.confirm(
        `Permanently delete user "${user.username}" and all their data? This cannot be undone.`,
      )
    ) {
      return;
    }
    setDeleting(true);
    setDeleteError(null);
    try {
      await adminDeleteUser(user.id);
      navigate('/admin/users');
    } catch (err: any) {
      setDeleteError(err.message || 'Failed to delete user.');
      setDeleting(false);
    }
  }

  useEffect(() => {
    getUser(id!).then((d) => setUser(d));
  }, [id]);

  if (!user) {
    return (
      <div className="container mx-auto px-4 py-8">
        <div className="animate-pulse bg-gray-100 rounded-lg p-8">
          Loading user details...
        </div>
      </div>
    );
  }

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="bg-white rounded-lg shadow-md p-6">
        <div className="flex justify-between items-center mb-6">
          <h1 className="text-2xl font-bold">User Details</h1>
          <div className="space-x-4">
            <Link
              to={`/admin/user/${user.id}/edit`}
              className="inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
            >
              Edit User
            </Link>
            <button
              onClick={handlePasswordReset}
              className="px-4 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700 transition-colors"
            >
              Send Password Reset
            </button>
            <button
              onClick={handleDelete}
              disabled={deleting}
              className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50"
            >
              {deleting ? 'Deleting...' : 'Delete User'}
            </button>
            {deleteError && (
              <p className="mt-2 text-sm text-red-600">{deleteError}</p>
            )}
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="space-y-4">
            <div className="flex items-center space-x-2">
              <span className="text-gray-600 font-medium w-32">Name:</span>
              <span className="text-gray-900">{user.username}</span>
            </div>
            <div className="flex items-center space-x-2">
              <span className="text-gray-600 font-medium w-32">ID:</span>
              <span className="text-gray-900">{user.id}</span>
            </div>
            <div className="flex items-center space-x-2">
              <span className="text-gray-600 font-medium w-32">Email:</span>
              <span className="text-gray-900">{user.email}</span>
            </div>
            <div className="flex items-center space-x-2">
              <span className="text-gray-600 font-medium w-32">
                Last Login:
              </span>
              <span className="text-gray-900">
                {user.last_login
                  ? new Date(user.last_login).toLocaleString()
                  : 'Never'}
              </span>
            </div>
            <div className="flex items-center space-x-2">
              <span className="text-gray-600 font-medium w-32">Last Seen:</span>
              <span className="text-gray-900">
                {user.last_seen
                  ? new Date(user.last_seen).toLocaleString()
                  : 'Never'}
              </span>
            </div>
          </div>

          <div className="space-y-4">
            <div className="flex items-center space-x-2">
              <span className="text-gray-600 font-medium w-32">
                Admin Status:
              </span>
              <span
                className={`px-2 py-1 rounded text-sm ${
                  user.is_admin
                    ? 'bg-purple-100 text-purple-800'
                    : 'bg-gray-100 text-gray-800'
                }`}
              >
                {user.is_admin ? 'Admin' : 'Regular User'}
              </span>
            </div>
            <div className="flex items-center space-x-2">
              <span className="text-gray-600 font-medium w-32">
                Account Status:
              </span>
              <span
                className={`px-2 py-1 rounded text-sm ${
                  user.is_active
                    ? 'bg-green-100 text-green-800'
                    : 'bg-red-100 text-red-800'
                }`}
              >
                {user.is_active ? 'Active' : 'Inactive'}
              </span>
            </div>
            <div className="flex items-center space-x-2">
              <span className="text-gray-600 font-medium w-32">
                Subscription:
              </span>
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
            </div>
            <div className="flex items-center space-x-2">
              <span className="text-gray-600 font-medium w-32">Stripe ID:</span>
              <a
                href={`https://dashboard.stripe.com/test/customers/${user.stripe_customer_id}`}
                className="text-blue-600 hover:text-blue-800 hover:underline"
                target="_blank"
                rel="noopener noreferrer"
              >
                {user.stripe_customer_id}
              </a>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
