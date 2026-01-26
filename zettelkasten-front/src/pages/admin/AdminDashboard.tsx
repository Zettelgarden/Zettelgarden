import React, { useState, useEffect } from "react";
import { getAdminStats, AdminStats } from "../../api/admin";
import { AdminErrorDisplay } from "../../components/admin/AdminErrorDisplay";

interface ErrorState {
  message: string;
  details?: string;
}

interface StatCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  icon?: string;
  color?: string;
}

function StatCard({ title, value, subtitle, icon = "📊", color = "bg-blue-500" }: StatCardProps) {
  return (
    <div className="bg-white rounded-lg shadow-md p-6 border-l-4 border-blue-500">
      <div className="flex items-center justify-between">
        <div className="flex-1">
          <p className="text-sm text-gray-600 font-medium">{title}</p>
          <p className="text-2xl font-bold text-gray-900 mt-1">{value}</p>
          {subtitle && (
            <p className="text-sm text-gray-500 mt-1">{subtitle}</p>
          )}
        </div>
        <div className="text-3xl ml-4">{icon}</div>
      </div>
    </div>
  );
}

interface StatSectionProps {
  title: string;
  icon: string;
  children: React.ReactNode;
}

function StatSection({ title, icon, children }: StatSectionProps) {
  return (
    <div className="mb-8">
      <h2 className="text-xl font-bold text-gray-800 mb-4 flex items-center">
        <span className="mr-2">{icon}</span>
        {title}
      </h2>
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {children}
      </div>
    </div>
  );
}

export function AdminDashboard() {
  const [stats, setStats] = useState<AdminStats | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<ErrorState | null>(null);

  useEffect(() => {
    const fetchStats = async () => {
      setIsLoading(true);
      setError(null);
      try {
        const data = await getAdminStats();
        setStats(data);
      } catch (err) {
        const message = err instanceof Error ? err.message : "Failed to load statistics";
        setError({ message, details: err instanceof Error ? err.stack : undefined });
      } finally {
        setIsLoading(false);
      }
    };
    fetchStats();
  }, []);

  if (isLoading) {
    return (
      <div className="container mx-auto px-4">
        <div className="space-y-6">
          <div className="animate-pulse">
            <div className="h-8 bg-gray-200 rounded w-1/3 mb-4"></div>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
              {Array.from({ length: 4 }).map((_, i) => (
                <div key={i} className="bg-gray-200 rounded-lg h-32"></div>
              ))}
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="container mx-auto px-4">
        <AdminErrorDisplay
          message={error.message}
          details={error.details}
          severity="error"
          onRetry={() => window.location.reload()}
          onDismiss={() => setError(null)}
        />
      </div>
    );
  }

  if (!stats) {
    return (
      <div className="container mx-auto px-4">
        <div className="text-center py-12 text-gray-500">
          No statistics available
        </div>
      </div>
    );
  }

  return (
    <div className="container mx-auto px-4">
      <h1 className="text-2xl sm:text-3xl font-bold text-gray-900 mb-6">
        Admin Dashboard
      </h1>

      {/* User Statistics */}
      <StatSection title="Users" icon="👥">
        <StatCard
          title="Total Users"
          value={stats.users.total}
          icon="👤"
        />
        <StatCard
          title="Active This Week"
          value={stats.users.active_this_week}
          subtitle={`${Math.round((stats.users.active_this_week / stats.users.total) * 100)}% of total`}
          icon="🟢"
        />
        <StatCard
          title="Active This Month"
          value={stats.users.active_this_month}
          subtitle={`${Math.round((stats.users.active_this_month / stats.users.total) * 100)}% of total`}
          icon="📅"
        />
        <StatCard
          title="New This Week"
          value={stats.users.new_this_week}
          subtitle={`+${stats.users.new_this_month} this month`}
          icon="✨"
        />
      </StatSection>

      {/* Subscription Statistics */}
      <StatSection title="Subscriptions" icon="💳">
        <StatCard
          title="Active & Trialing"
          value={stats.subscriptions.active}
          subtitle="Paying users"
          icon="✅"
        />
        <StatCard
          title="Free Users"
          value={stats.subscriptions.free}
          subtitle={`${Math.round((stats.subscriptions.free / stats.subscriptions.total) * 100)}% of total`}
          icon="🆓"
        />
        <StatCard
          title="Past Due"
          value={stats.subscriptions.past_due}
          subtitle="Need attention"
          icon="⚠️"
        />
        <StatCard
          title="Total Subscriptions"
          value={stats.subscriptions.total}
          icon="📊"
        />
      </StatSection>

      {/* Revenue Statistics */}
      <StatSection title="Revenue" icon="💰">
        <StatCard
          title="Total Revenue"
          value={`$${stats.revenue.total_revenue.toFixed(2)}`}
          icon="💵"
        />
        <StatCard
          title="This Month"
          value={`$${stats.revenue.revenue_this_month.toFixed(2)}`}
          icon="📈"
        />
        <StatCard
          title="Monthly Recurring"
          value={`$${stats.revenue.monthly_recurring_revenue.toFixed(2)}`}
          subtitle="MRR"
          icon="🔄"
        />
        <StatCard
          title="Avg Revenue/User"
          value={`$${stats.revenue.total_revenue > 0 ? (stats.revenue.monthly_recurring_revenue / stats.subscriptions.active).toFixed(2) : '0.00'}`}
          subtitle="Per paying user"
          icon="📊"
        />
      </StatSection>

      {/* Content Statistics */}
      <StatSection title="Content" icon="📝">
        <StatCard
          title="Total Cards"
          value={stats.content.total_cards}
          subtitle="Zettelkasten notes"
          icon="📇"
        />
        <StatCard
          title="Total Tasks"
          value={stats.content.total_tasks}
          subtitle="Active tasks"
          icon="✅"
        />
        <StatCard
          title="Total Files"
          value={stats.content.total_files}
          subtitle="Uploaded files"
          icon="📎"
        />
        <StatCard
          title="Chat Messages"
          value={stats.content.total_chat_messages}
          subtitle="AI conversations"
          icon="💬"
        />
      </StatSection>

      {/* Entities and Facts (PRO features) */}
      {(stats.content.total_entities > 0 || stats.content.total_facts > 0) && (
        <StatSection title="PRO Features" icon="⭐">
          <StatCard
            title="Total Entities"
            value={stats.content.total_entities}
            subtitle="Named entities"
            icon="🏷️"
          />
          <StatCard
            title="Total Facts"
            value={stats.content.total_facts}
            subtitle="Structured data"
            icon="🧠"
          />
        </StatSection>
      )}
    </div>
  );
}
