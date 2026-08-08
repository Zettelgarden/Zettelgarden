# MobileTopBar Quick Start Guide

## Import

```tsx
import { MobileTopBar } from '@/components/layout';
// or
import { MobileTopBar } from '@/components/layout/MobileTopBar';
```

## Basic Usage

```tsx
<MobileTopBar title="Page Title" />
```

## Common Patterns

### Navigation Page (Back Button)

```tsx
<MobileTopBar title="Details" onBack={() => navigate(-1)} />
```

### List Page (Menu + Badge)

```tsx
<MobileTopBar
  title="Inbox"
  badge={unreadCount}
  onMenuClick={() => setShowSidebar(true)}
/>
```

### Action Page (Back + Action Button)

```tsx
<MobileTopBar
  title="Edit"
  onBack={() => navigate(-1)}
  actions={<button onClick={handleSave}>Save</button>}
/>
```

### Multiple Actions

```tsx
<MobileTopBar
  title="Messages"
  onBack={() => navigate(-1)}
  actions={
    <>
      <button onClick={handleSearch}>Search</button>
      <button onClick={handleMore}>More</button>
    </>
  }
/>
```

## Props Reference

| Prop        | Type             | Required | Default   |
| ----------- | ---------------- | -------- | --------- |
| title       | string           | ✅       | -         |
| badge       | string \| number | ❌       | undefined |
| onBack      | () => void       | ❌       | undefined |
| onMenuClick | () => void       | ❌       | undefined |
| actions     | ReactNode        | ❌       | undefined |
| className   | string           | ❌       | ""        |
| zIndex      | number           | ❌       | 40        |
| mobileOnly  | boolean          | ❌       | true      |

## Badge Behavior

Badge is shown when:

- ✅ `badge={5}` - Shows "5"
- ✅ `badge="New"` - Shows "New"
- ✅ `badge={99}` - Shows "99"
- ✅ `badge={150}` - Shows "150"

Badge is hidden when:

- ❌ `badge={0}` - Hidden
- ❌ `badge=""` - Hidden
- ❌ Not provided - Hidden

## Button Priority

Left button logic:

1. If `onBack` provided → Show back button
2. Else if `onMenuClick` provided → Show menu button
3. Else → No left button

## Styling Tips

### Add Shadow

```tsx
<MobileTopBar title="Modal" className="shadow-md" />
```

### Higher Z-Index

```tsx
<MobileTopBar title="Overlay" zIndex={50} />
```

### Show on Desktop

```tsx
<MobileTopBar title="Always" mobileOnly={false} />
```

## Action Button Patterns

### Icon Button

```tsx
<button
  className="p-2 -mr-2 text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg"
  aria-label="Settings"
>
  <SettingsIcon />
</button>
```

### Text Button

```tsx
<button className="p-2 -mr-2 text-blue-600 font-medium text-sm">Save</button>
```

### Primary Button

```tsx
<button className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700">
  Action
</button>
```

## Full Example

```tsx
import { MobileTopBar } from '@/components/layout';
import { useNavigate } from 'react-router-dom';

function MyPage() {
  const navigate = useNavigate();
  const [unreadCount, setUnreadCount] = useState(5);

  return (
    <div>
      <MobileTopBar
        title="My Page"
        badge={unreadCount > 0 ? unreadCount : undefined}
        onBack={() => navigate(-1)}
        zIndex={50}
        actions={
          <div className="flex items-center gap-1">
            <button onClick={handleRefresh} aria-label="Refresh">
              <RefreshIcon />
            </button>
            <button onClick={handleSave} className="text-blue-600 font-medium">
              Save
            </button>
          </div>
        }
      />
      {/* Page content */}
    </div>
  );
}
```

## Migration from Existing Components

### From RssMobileTopBar

Before:

```tsx
<RssMobileTopBar
  title="RSS"
  unreadCount={5}
  onMenuClick={handleMenu}
  onSettingsClick={handleSettings}
/>
```

After:

```tsx
<MobileTopBar
  title="RSS"
  badge={5}
  onMenuClick={handleMenu}
  actions={<SettingsButton onClick={handleSettings} />}
/>
```

## Troubleshooting

### Badge not showing

Ensure badge is not `0`, `""`, or `undefined`.

### Button not showing

Ensure either `onBack` or `onMenuClick` is provided.

### Actions not visible

Check that `actions` prop contains valid React elements.

### Z-index conflicts

Increase `zIndex` prop if other elements cover the top bar.
