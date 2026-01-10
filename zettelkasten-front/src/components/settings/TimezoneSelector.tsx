import React from "react";

interface TimezoneSelectorProps {
  value: string;
  onChange: (timezone: string) => void;
}

export function TimezoneSelector({ value, onChange }: TimezoneSelectorProps) {
  // Get all available timezones using Intl API
  const timezones = Intl.supportedValuesOf('timeZone');

  // Group timezones by region for better organization
  const timezonesByRegion = timezones.reduce((acc, tz) => {
    const parts = tz.split('/');
    const region = parts.length > 1 ? parts[0] : 'Other';
    if (!acc[region]) {
      acc[region] = [];
    }
    acc[region].push(tz);
    return acc;
  }, {} as Record<string, string[]>);

  // Sort regions and timezones
  const sortedRegions = Object.keys(timezonesByRegion).sort();

  return (
    <div className="space-y-2">
      <label className="block text-sm font-medium text-gray-700">
        Time Zone
      </label>
      <select
        value={value || "UTC"}
        onChange={(e) => onChange(e.target.value)}
        className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md"
      >
        <option value="UTC">UTC (Coordinated Universal Time)</option>
        {sortedRegions.map((region) => (
          <optgroup key={region} label={region}>
            {timezonesByRegion[region].sort().map((tz) => {
              // Format the timezone display name
              const displayName = tz.replace(/_/g, ' ');
              return (
                <option key={tz} value={tz}>
                  {displayName}
                </option>
              );
            })}
          </optgroup>
        ))}
      </select>
      <p className="mt-1 text-sm text-gray-500">
        Select your default time zone. This will be used for displaying and scheduling tasks.
      </p>
    </div>
  );
}
