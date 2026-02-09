# MobileTopBar Component

A generic, reusable mobile top bar component for responsive pages in Zettelgarden.

## Features

- **Sticky positioning**: Stays at the top of the viewport when scrolling
- **Flexible layout**: Left action, center title, right actions
- **Badge support**: Display badges (counts, labels) next to the title
- **Multiple left actions**: Back button, menu button, or custom actions
- **Multiple right actions**: Support for single or multiple action buttons
- **Responsive**: Hidden on desktop (md+) by default, can be configured
- **Accessible**: Proper ARIA labels and semantic HTML
- **Customizable**: Configurable z-index, className, and mobile-only behavior

## Installation

The component is located at `src/components/layout/MobileTopBar.tsx`.

## Usage

### Basic Usage

```tsx
import { MobileTopBar } from "@/components/layout";

function MyPage() {
  return (
    <>
      <MobileTopBar title="My Page" />
      {/* Page content */}
    </>
  );
}
```

### With Back Button

```tsx
<MobileTopBar
  title="Article Details"
  onBack={() => navigate(-1)}
/>
```

### With Menu Button

```tsx
<MobileTopBar
  title="RSS Feeds"
  onMenuClick={() => setShowMenu(true)}
/>
```

### With Badge

```tsx
<MobileTopBar
  title="Notifications"
  badge={5}
  onMenuClick={() => setShowMenu(true)}
/>
```

### With Action Buttons

```tsx
<MobileTopBar
  title="Edit Profile"
  onBack={() => navigate(-1)}
  actions={
    <button
      onClick={handleSave}
      className="p-2 -mr-2 text-blue-600 hover:text-blue-800 hover:bg-blue-50 rounded-lg font-medium text-sm"
    >
      Save
    </button>
  }
/>
```

### With Multiple Actions

```tsx
<MobileTopBar
  title="Messages"
  badge="3 new"
  onBack={() => navigate(-1)}
  actions={
    <div className="flex items-center gap-1">
      <button onClick={handleSearch} aria-label="Search">
        <SearchIcon />
      </button>
      <button onClick={handleMore} aria-label="More options">
        <MoreIcon />
      </button>
    </div>
  }
/>
```

### Custom Styling

```tsx
<MobileTopBar
  title="Modal Header"
  onBack={() => closeModal()}
  zIndex={50}
  className="shadow-md"
  actions={<DoneButton />}
/>
```

### Visible on All Screen Sizes

```tsx
<MobileTopBar
  title="Always Visible"
  mobileOnly={false}
  actions={<ActionButton />}
/>
```

## Props

### MobileTopBarProps

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `title` | `string` | *required* | The title to display in the top bar |
| `badge` | `string \| number` | `undefined` | Optional badge text or number to display next to the title |
| `onBack` | `() => void` | `undefined` | Callback for the back button (shows back button when provided) |
| `onMenuClick` | `() => void` | `undefined` | Callback for the menu button (shows menu button when onBack is not provided) |
| `actions` | `ReactNode` | `undefined` | Action buttons or elements to display on the right side |
| `className` | `string` | `""` | Optional custom class name for additional styling |
| `zIndex` | `number` | `40` | Z-index value for the top bar |
| `mobileOnly` | `boolean` | `true` | Whether to show the component only on mobile devices |

## Design Patterns

### Patterns Found in Existing Code

1. **RSS Page Pattern** (`src/pages/RssPage.tsx`):
   - Left: Menu button
   - Center: Title with unread badge
   - Right: Settings button
   - Sticky positioning with z-index 40

2. **Mobile Reader Pattern** (`src/components/rss/RssMobileReader.tsx`):
   - Left: Back button
   - Center: Title with flex-1 and truncate
   - Right: Action button (Convert)
   - Sticky positioning with shadow

3. **Admin Top Bar Pattern** (`src/components/AdminTopBar.tsx`):
   - Desktop-only variant
   - Simple title and action button layout

## Styling

The component uses Tailwind CSS classes:

- **Container**: `sticky top-0 bg-white border-b border-gray-200 px-4 py-3 flex items-center justify-between`
- **Mobile hidden**: `md:hidden` (when `mobileOnly` is true)
- **Buttons**: `p-2 -ml-2` or `p-2 -mr-2` for proper spacing
- **Hover states**: `hover:text-gray-900 hover:bg-gray-100 rounded-lg`
- **Badge**: `bg-red-500 text-white text-xs font-bold px-2 py-0.5 rounded-full`

## Accessibility

- All buttons include `aria-label` attributes
- Semantic HTML structure
- Proper focus states with hover and active styles
- Badge text is properly formatted for screen readers

## Examples

See `MobileTopBar.examples.tsx` for more usage examples.

## Future Enhancements

Possible future additions:
- Search input variant
- Progress indicator variant
- Breadcrumb navigation variant
- Custom left/right slot components
- Animated transitions
- Theme variants (dark mode)

## Related Components

- `RssMobileTopBar` - RSS-specific mobile top bar (can be refactored to use this component)
- `RssMobileReader` - Mobile reader with inline top bar
- `AdminTopBar` - Desktop admin top bar
